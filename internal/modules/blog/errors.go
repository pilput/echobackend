package blog

import "errors"

// Blog domain errors.
var (
	ErrPostNotFound           = errors.New("post not found")
	ErrPostSpamRejected       = errors.New("post rejected by anti-spam policy")
	ErrAlreadyLiked           = errors.New("user has already liked this post")
	ErrNotLiked               = errors.New("user has not liked this post")
	ErrCommentNotFound        = errors.New("comment not found")
	ErrCommentNotOwned        = errors.New("not authorized to modify this comment")
	ErrPostNotOwned           = errors.New("not authorized to modify this post")
	ErrInvalidPostID          = errors.New("invalid post ID format")
	ErrEmptyPostID            = errors.New("post ID cannot be empty")
	ErrCannotReportOwnPost    = errors.New("cannot report your own post")
	ErrNoPendingReports       = errors.New("post has no pending reports")
	ErrInvalidReportStatus    = errors.New("invalid report status")
	ErrPostAlreadyHidden      = errors.New("post is already hidden")
	ErrPostNotHidden          = errors.New("post is not hidden")
	ErrTagNotFound            = errors.New("tag not found")
	ErrTagNameRequired        = errors.New("tag name is required")
	ErrTagNameEmpty           = errors.New("tag name cannot be empty")
	ErrInvalidTagID           = errors.New("invalid tag ID")
	ErrFileTooLarge           = errors.New("file size must not exceed 1 MB")
	ErrBookmarkNotFound       = errors.New("bookmark not found")
	ErrBookmarkFolderNotFound = errors.New("bookmark folder not found")
)
