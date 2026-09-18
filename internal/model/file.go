package model

import (
	"time"

	"gorm.io/gorm"
)

type File struct {
	ID        string         `json:"id" gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt *time.Time     `json:"created_at"`
	UpdatedAt *time.Time     `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Name      *string        `json:"name" gorm:"type:varchar(255)"`
	Path      *string        `json:"path"`
	Size      *int           `json:"size"`
	Type      *string        `json:"type" gorm:"type:varchar(255)"`
	CreatedBy *string        `json:"created_by" gorm:"type:uuid"`
	User      *User          `gorm:"foreignKey:CreatedBy"`
}

func (File) TableName() string {
	return "files"
}
