package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dockyard/config"
	"dockyard/models"
	"dockyard/services/crypto"
	dockerSvc "dockyard/services/docker"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseEnvVars(raw string) []string {
	var vars []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, "=") {
			vars = append(vars, line)
		}
	}
	return vars
}

// mountLineRe matches "host:container[:ro]" where the host path is either
// POSIX-absolute (/...) or Windows-absolute (C:\... or C:/...). The backend
// always runs as a Linux binary (Alpine container), so Go's path/filepath
// only understands POSIX syntax regardless of the Docker host's own OS -
// without this, any Windows-style path typed by a Windows user (and the
// drive-letter colon it contains) would be misparsed and silently rejected.
var mountLineRe = regexp.MustCompile(`^([A-Za-z]:[\\/][^:]*|/[^:]*):([^:]+)(?::(ro))?$`)

// normalizeHostPath makes a host-side mount path comparable regardless of
// whether it was typed with POSIX or Windows path syntax: backslashes are
// converted to forward slashes, a trailing slash is dropped, and a leading
// Windows drive letter is lowercased (Windows drive letters and, in
// practice, the whole path are case-insensitive; a POSIX path has no drive
// letter so this step is a no-op for it).
func normalizeHostPath(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	for len(p) > 1 && strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}
	if len(p) >= 2 && p[1] == ':' && ((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) {
		p = strings.ToLower(p[:1]) + p[1:]
	}
	return p
}

// validateMounts parses a multi-line string of bind mounts in "host_path:container_path[:ro]"
// format (host_path may be POSIX-style, e.g. /home/user/x, or Windows-style, e.g. C:\Users\x or
// C:/Users/x) and returns only those whose host path resolves inside allowedBase. Any mount that
// is not absolute, contains "..", or points outside the project's sandbox directory is rejected.
// This prevents a user from mounting arbitrary host paths (e.g. "/" or the Docker socket).
func validateMounts(raw, allowedBase string) []string {
	var binds []string
	if allowedBase == "" {
		// Without a known-safe base we cannot confine mounts, so allow none.
		return binds
	}
	base := normalizeHostPath(allowedBase)

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := mountLineRe.FindStringSubmatch(line)
		if m == nil {
			log.Printf("rejected unsafe mount (not absolute / malformed): %s", line)
			continue
		}
		rawHostPath := m[1]

		if strings.Contains(rawHostPath, "..") {
			log.Printf("rejected unsafe mount (contains ..): %s", line)
			continue
		}

		hostPath := normalizeHostPath(rawHostPath)
		if hostPath != base && !strings.HasPrefix(hostPath, base+"/") {
			log.Printf("rejected mount outside allowed dir %q: %s", base, line)
			continue
		}
		binds = append(binds, line)
	}
	return binds
}

// detectScheme returns "https" if CERT_FILE and KEY_FILE are both set in envVars.
func detectScheme(envVars string) string {
	hasCert := strings.Contains(envVars, "CERT_FILE=")
	hasKey := strings.Contains(envVars, "KEY_FILE=")
	if hasCert && hasKey {
		return "https"
	}
	return "http"
}

func buildURL(scheme string, port int) string {
	return fmt.Sprintf("%s://localhost:%d", scheme, port)
}

func LaunchContainer(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		projectID := c.Param("id")

		// Atomically claim the project: only update if currently stopped/error.
		result := db.Model(&models.Project{}).
			Where("id = ? AND user_id = ? AND status IN ('stopped', 'error')", projectID, userID).
			Update("status", "pulling")

		if result.Error != nil {
			log.Printf("failed to claim project for launch: %v", result.Error)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if result.RowsAffected == 0 {
			var project models.Project
			if err := db.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
				return
			}
			c.JSON(http.StatusConflict, gin.H{
				"status":  project.Status,
				"message": fmt.Sprintf("project is already %s", project.Status),
			})
			return
		}

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		registryPassword, err := crypto.Decrypt(cfg.EncryptionKey, project.RegistryPassword)
		if err != nil {
			log.Printf("failed to decrypt registry password for project %s: %v", project.ID, err)
			db.Model(&project).Update("status", "error")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		scheme := project.Scheme
		if scheme == "" {
			scheme = detectScheme(project.EnvVars)
		}

		go func() {
			// Build bind mounts: prefer the Mounts field; fall back to legacy host: prefix in Repository.
			// Both are user-controlled, so every host path is validated against the project's sandbox.
			var allowedBase string
			if cfg.FilesHostDir != "" {
				allowedBase = filepath.Join(cfg.FilesHostDir, project.ID.String())
			}

			var binds []string
			if project.Mounts != "" {
				binds = validateMounts(project.Mounts, allowedBase)
			} else if strings.HasPrefix(project.Repository, "host:") {
				hostPath := strings.TrimPrefix(project.Repository, "host:")
				binds = validateMounts(fmt.Sprintf("%s:/app", hostPath), allowedBase)
			}

			// Auto-mount uploaded files directory if it exists and FILES_HOST_DIR is configured.
			if cfg.FilesHostDir != "" {
				hostFilesDir := filepath.Join(cfg.FilesHostDir, project.ID.String())
				if _, err := os.Stat(filepath.Join(cfg.FilesDir, project.ID.String())); err == nil {
					binds = append(binds, fmt.Sprintf("%s:/dockyard-files", hostFilesDir))
				}
			}

			if err := dockerSvc.PullImage(project.DockerImage, project.RegistryUser, registryPassword); err != nil {
				log.Printf("failed to pull image for project %s: %v", project.ID, err)
				db.Model(&project).Update("status", "error")
				return
			}

			containerID, port, err := dockerSvc.RunContainer(
				project.DockerImage,
				parseEnvVars(project.EnvVars),
				binds,
				project.ContainerPort,
				project.TerminalMode,
			)
			if err != nil {
				log.Printf("failed to start container for project %s: %v", project.ID, err)
				db.Model(&project).Update("status", "error")
				return
			}

			now := time.Now()
			db.Model(&project).Updates(map[string]interface{}{
				"status":           "running",
				"scheme":           scheme,
				"container_id":     containerID,
				"port":             port,
				"started_at":       now,
				"last_accessed_at": now,
			})
		}()

		c.JSON(http.StatusAccepted, gin.H{"message": "launch started", "status": "pulling"})
	}
}

func StopContainer(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		if project.ContainerID != nil {
			if err := dockerSvc.StopAndRemoveContainer(*project.ContainerID); err != nil {
				log.Printf("failed to stop container %s: %v", *project.ContainerID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to stop container: " + err.Error()})
				return
			}
		}

		db.Model(&project).Updates(map[string]interface{}{
			"status":       "stopped",
			"container_id": nil,
			"port":         nil,
		})

		c.JSON(http.StatusOK, gin.H{"message": "container stopped"})
	}
}

func GetStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		resp := gin.H{
			"status": project.Status,
			"port":   project.Port,
		}
		if project.Port != nil && *project.Port > 0 {
			scheme := project.Scheme
			if scheme == "" {
				scheme = "http"
			}
			resp["url"] = buildURL(scheme, *project.Port)
		}

		c.JSON(http.StatusOK, resp)
	}
}
