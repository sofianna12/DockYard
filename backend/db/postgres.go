package db

import (
	"log"
	"time"

	"dockyard/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string, autoMigrate bool) *gorm.DB {
	var database *gorm.DB
	var err error

	for i := 0; i < 10; i++ {
		database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Database connection attempt %d failed: %v. Retrying...", i+1, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// In production the SQL files in database/init/ are the source of truth.
	// AutoMigrate is a dev convenience that keeps the schema in sync with the
	// models without writing a migration; disable it via AUTO_MIGRATE=false.
	if autoMigrate {
		if err := database.AutoMigrate(&models.User{}, &models.Project{}); err != nil {
			log.Fatal("AutoMigrate failed:", err)
		}
		log.Println("AutoMigrate complete")
	} else {
		log.Println("AutoMigrate skipped (AUTO_MIGRATE=false)")
	}

	log.Println("Database connected")
	return database
}
