package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"dockyard/config"
	"dockyard/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxUploadSize = 10 << 20 // 10 MB

// safeFilename rejects any filename that could escape the project directory.
func safeFilename(name string) bool {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || name == ".." {
		return false
	}
	clean := filepath.Clean(name)
	return clean == name && !strings.HasPrefix(clean, ".")
}

func projectFilesDir(cfg *config.Config, projectID string) string {
	return filepath.Join(cfg.FilesDir, projectID)
}

func UploadFile(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
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

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file required (max 10 MB)"})
			return
		}
		defer file.Close()

		if !safeFilename(header.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
			return
		}

		dir := projectFilesDir(cfg, project.ID.String())
		if err := os.MkdirAll(dir, 0750); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create storage directory"})
			return
		}

		dest := filepath.Join(dir, header.Filename)
		out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
		defer out.Close()

		if _, err := io.Copy(out, file); err != nil {
			os.Remove(dest)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"filename":       header.Filename,
			"container_path": "/dockyard-files/" + header.Filename,
		})
	}
}

func ListFiles(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
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

		dir := projectFilesDir(cfg, project.ID.String())
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusOK, []gin.H{})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
			return
		}

		type fileInfo struct {
			Filename      string `json:"filename"`
			ContainerPath string `json:"container_path"`
			SizeBytes     int64  `json:"size_bytes"`
		}

		result := make([]fileInfo, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			result = append(result, fileInfo{
				Filename:      e.Name(),
				ContainerPath: "/dockyard-files/" + e.Name(),
				SizeBytes:     size,
			})
		}

		c.JSON(http.StatusOK, result)
	}
}

func DeleteFile(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
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

		filename := c.Param("filename")
		if !safeFilename(filename) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
			return
		}

		path := filepath.Join(projectFilesDir(cfg, project.ID.String()), filename)
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

