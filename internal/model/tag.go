package model

import (
	"time"
)

type Tag struct {
	ID        uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string     `json:"name" gorm:"uniqueIndex:idx_tags_name;type:varchar(30);not null"`
	CreatedAt *time.Time `json:"created_at"`
	Posts     []Post     `gorm:"many2many:posts_to_tags;"`
}

func (Tag) TableName() string {
	return "tags"
}
