package blog

// ensurePostInteractable rejects engagement (like, comment, bookmark, view,
// report) on a post a moderator has hidden. It is reported as "not found" so
// the API does not confirm that a hidden post exists. Owner and admin lookups
// by ID deliberately skip this check.
func ensurePostInteractable(post *Post) error {
	if post != nil && post.HiddenAt != nil {
		return ErrPostNotFound
	}
	return nil
}
