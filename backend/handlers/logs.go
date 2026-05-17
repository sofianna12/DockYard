package handlers

import (
	"bytes"
	"net/http"

	"dockyard/models"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	dockerSvc "dockyard/services/docker"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetLogs(db *gorm.DB) gin.HandlerFunc {
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

		if project.ContainerID == nil {
			c.JSON(http.StatusOK, gin.H{"logs": ""})
			return
		}

		tail := c.DefaultQuery("tail", "100")

		cli := dockerSvc.GetClient()
		reader, err := cli.ContainerLogs(c.Request.Context(), *project.ContainerID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       tail,
			Timestamps: true,
		})
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"logs": "Container logs unavailable: " + err.Error()})
			return
		}
		defer reader.Close()

		var buf bytes.Buffer
		stdcopy.StdCopy(&buf, &buf, reader)

		c.JSON(http.StatusOK, gin.H{"logs": buf.String()})
	}
}
