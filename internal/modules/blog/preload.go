package blog

import "gorm.io/gorm"

// activePostUserJoin restricts a posts query to authors that are not deleted.
func activePostUserJoin(db *gorm.DB) *gorm.DB {
	return db.Joins("JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL")
}
