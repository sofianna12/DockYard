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
		if err := dockerSvc.StopAndRemoveContainer(*p.ContainerID); err != nil {
			exists, running, inspectErr := dockerSvc.ContainerRunning(*p.ContainerID)
			if inspectErr != nil {
				log.Printf("cleanup: failed to stop container for project %s (%v), and could not verify its actual state (%v); leaving status unchanged for retry", p.Title, err, inspectErr)
				continue
			}
			if exists && running {
				log.Printf("cleanup: failed to stop container for project %s (%v); container is still running (zombie) - leaving status as running for retry next cycle", p.Title, err)
				continue
			}
			log.Printf("cleanup: StopAndRemoveContainer reported an error for project %s (%v), but the daemon confirms the container no longer exists - proceeding to mark stopped", p.Title, err)
		}
		db.Model(&p).Updates(map[string]interface{}{
			"status":       "stopped",
			"container_id": nil,
			"port":         nil,
		})
	}

	reconcileOrphanedRunning(db)
}


func reconcileOrphanedRunning(db *gorm.DB) {
	result := db.Model(&models.Project{}).
		Where("status = ? AND container_id IS NULL", "running").
		Update("status", "error")
	if result.RowsAffected > 0 {
		log.Printf("cleanup: reconciled %d orphaned project(s) stuck in status=running with no container_id", result.RowsAffected)
	}
}
