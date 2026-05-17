package cleanup

import (
	"log"
	"time"

	"dockyard/models"
	dockerSvc "dockyard/services/docker"

	"gorm.io/gorm"
)

func StartWorker(db *gorm.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	log.Println("Cleanup worker started")

	for range ticker.C {
		runCleanup(db)
	}
}

func runCleanup(db *gorm.DB) {
	var projects []models.Project

	db.Raw(`
		SELECT * FROM projects
		WHERE status = 'running'
		AND last_accessed_at < NOW() - (auto_stop_min || ' minutes')::INTERVAL
		AND auto_stop_min > 0
	`).Scan(&projects)

	for _, p := range projects {
		if p.ContainerID == nil {
			continue
		}
		log.Printf("Stopping idle container for project: %s", p.Title)
		dockerSvc.StopAndRemoveContainer(*p.ContainerID)
		db.Model(&p).Updates(map[string]interface{}{
			"status":       "stopped",
			"container_id": nil,
			"port":         nil,
		})
	}
}
