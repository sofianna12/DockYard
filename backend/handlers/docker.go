package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

// parseMounts parses a multi-line string of bind mounts in "host_path:container_path" format.
func parseMounts(raw string) []string {
	var binds []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, ":") {
			binds = append(binds, line)
		}
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

		scheme := detectScheme(project.EnvVars)

		go func() {
			// Build bind mounts: prefer the Mounts field; fall back to legacy host: prefix in Repository.
			var binds []string
			if project.Mounts != "" {
				binds = parseMounts(project.Mounts)
			} else if strings.HasPrefix(project.Repository, "host:") {
				hostPath := strings.TrimPrefix(project.Repository, "host:")
				binds = []string{fmt.Sprintf("%s:/app", hostPath)}
			}

			// Auto-mount uploaded files directory if it exists and FILES_HOST_DIR is configured.
			if cfg.FilesHostDir != "" {
				hostFilesDir := filepath.Join(cfg.FilesHostDir, project.ID.String())
				if _, err := os.Stat(filepath.Join(cfg.FilesDir, project.ID.String())); err == nil {
					binds = append(binds, fmt.Sprintf("%s:/dockyard-files:ro", hostFilesDir))
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
