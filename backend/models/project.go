package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Title            string     `gorm:"not null" json:"title"`
	Description      string     `json:"description"`
	Repository       string     `json:"repository"`
	DockerImage      string     `gorm:"not null" json:"docker_image"`
	RegistryUser     string     `gorm:"not null;default:''" json:"registry_user"`
	RegistryPassword string     `gorm:"not null;default:''" json:"-"`
	EnvVars          string     `gorm:"not null;default:''" json:"env_vars"`
	Mounts           string     `gorm:"not null;default:''" json:"mounts"`
	ContainerPort    int        `gorm:"default:0" json:"container_port"`
	TerminalMode     bool       `gorm:"default:false" json:"terminal_mode"`
	WebTerminal      bool       `gorm:"default:false" json:"web_terminal"`
	AutoStopMin      int        `gorm:"default:60" json:"auto_stop_min"`
	Status           string     `gorm:"default:'stopped'" json:"status"` // stopped | pulling | running | error
	Scheme           string     `gorm:"default:'http'" json:"scheme"`    // http | https
	ContainerID      *string    `json:"container_id"`
	Port             *int       `json:"port"`
	StartedAt        *time.Time `json:"started_at"`
	LastAccessedAt   *time.Time `json:"last_accessed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
