package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"dockyard/config"
	"dockyard/models"
	dockerSvc "dockyard/services/docker"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsResizeMsg struct {
	Type string `json:"type"`
	Rows uint   `json:"rows"`
	Cols uint   `json:"cols"`
}

func TerminalWS(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Auth via query param (WebSocket can't set headers from browser JS)
		tokenStr := c.Query("token")
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		userIDStr, _ := claims["user_id"].(string)

		projectID := c.Param("id")

		var project models.Project
		if err := db.Where("id = ? AND user_id = ?", projectID, userIDStr).First(&project).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}

		if project.Status != "running" || project.ContainerID == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "container not running"})
			return
		}

		if !project.TerminalMode && !project.WebTerminal {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not a terminal project"})
			return
		}

		exec, err := dockerSvc.ExecTTY(*project.ContainerID)
		if err != nil {
			log.Printf("exec failed for project %s: %v", project.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach terminal"})
			return
		}
		defer exec.Conn.Close()

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}
		defer ws.Close()

		done := make(chan struct{})

		// exec stdout → ws
		go func() {
			defer close(done)
			buf := make([]byte, 4096)
			for {
				n, err := exec.Reader.Read(buf)
				if n > 0 {
					if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}()

		// ws → exec stdin (or resize events)
		for {
			msgType, data, err := ws.ReadMessage()
			if err != nil {
				return
			}

			if msgType == websocket.TextMessage {
				var msg wsResizeMsg
				if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" && msg.Rows > 0 && msg.Cols > 0 {
					dockerSvc.ResizeExecTTY(exec.ID, msg.Rows, msg.Cols)
				}
				continue
			}

			if _, err := io.Writer(exec.Conn).Write(data); err != nil {
				return
			}
		}
	}
}
