package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides a UUID primary key and timestamp fields for all GORM models.
type BaseModel struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;column:id"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime:milli;column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// BeforeCreate assigns a new UUID to ID if one has not been set.
func (m *BaseModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
