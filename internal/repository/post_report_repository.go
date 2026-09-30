package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostReportRepository interface {
	// Create stores the report. If the same user already has a pending report
	// on the post, the new one is silently ignored.
	Create(ctx context.Context, report *model.PostReport) error
	ListGroupedByPost(ctx context.Context, status string, limit, offset int) ([]dto.PostReportGroup, int64, error)
	ListByPost(ctx context.Context, postID, status string, limit, offset int) ([]*model.PostReport, int64, error)
	// Moderate applies one admin decision atomically: it writes the audit row,
	// flips posts.hidden_at for hide/unhide, and closes the post's pending
	// reports for hide/dismiss.
	Moderate(ctx context.Context, postID, adminID, action string, note *string) error
	ListActionsByPost(ctx context.Context, postID string, limit, offset int) ([]*model.PostModerationAction, int64, error)
}

type postReportRepository struct {
	db *gorm.DB
}

func NewPostReportRepository(db *gorm.DB) PostReportRepository {
	return &postReportRepository{db: db}
}

func (r *postReportRepository) Create(ctx context.Context, report *model.PostReport) error {
	// The conflict target must repeat the partial index predicate, otherwise
	// Postgres cannot infer which index to arbitrate on.
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:     []clause.Column{{Name: "post_id"}, {Name: "reporter_id"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "status = 'pending'"}}},
			DoNothing:   true,
		}).
		Create(report).Error
}

func (r *postReportRepository) ListGroupedByPost(ctx context.Context, status string, limit, offset int) ([]dto.PostReportGroup, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.PostReport{}).
		Where("status = ?", status).
		Distinct("post_id").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count reported posts: %w", err)
	}

	var groups []dto.PostReportGroup
	// Table() opts out of the soft-delete scope, so users.deleted_at is written
	// by hand (posts has no soft delete). The author join is LEFT so a post whose
	// author was deleted still shows up in the queue.
	err := r.db.WithContext(ctx).
		Table("post_reports").
		Select(`posts.id AS post_id, posts.title AS post_title, posts.slug AS post_slug,
			users.username AS author_username, posts.hidden_at,
			COUNT(*) AS report_count,
			MODE() WITHIN GROUP (ORDER BY post_reports.reason) AS top_reason,
			MAX(post_reports.created_at) AS last_reported_at`).
		Joins("JOIN posts ON posts.id = post_reports.post_id").
		Joins("LEFT JOIN users ON users.id = posts.created_by AND users.deleted_at IS NULL").
		Where("post_reports.status = ?", status).
		Group("posts.id, posts.title, posts.slug, users.username, posts.hidden_at").
		Order("COUNT(*) DESC, MAX(post_reports.created_at) DESC").
		Limit(limit).
		Offset(offset).
		Scan(&groups).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list reported posts: %w", err)
	}
	return groups, total, nil
}

func (r *postReportRepository) ListByPost(ctx context.Context, postID, status string, limit, offset int) ([]*model.PostReport, int64, error) {
	build := func() *gorm.DB {
		q := r.db.WithContext(ctx).Model(&model.PostReport{}).Where("post_reports.post_id = ?", postID)
		if status != "" {
			q = q.Where("post_reports.status = ?", status)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count post reports: %w", err)
	}

	var reports []*model.PostReport
	if err := build().
		Preload("Reporter", preloadUserBrief).
		Preload("Action").
		Order("post_reports.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reports).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list post reports: %w", err)
	}
	return reports, total, nil
}

func (r *postReportRepository) Moderate(ctx context.Context, postID, adminID, action string, note *string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the post row so two admins acting at once serialize instead of
		// both passing the state check below.
		var post struct {
			CreatedBy *string
			HiddenAt  *time.Time
		}
		err := tx.Table("posts").
			Select("created_by, hidden_at").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", postID).
			Take(&post).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrPostNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to load post: %w", err)
		}

		switch action {
		case model.ModerationActionHide:
			if post.HiddenAt != nil {
				return apperrors.ErrPostAlreadyHidden
			}
		case model.ModerationActionUnhide:
			if post.HiddenAt == nil {
				return apperrors.ErrPostNotHidden
			}
		case model.ModerationActionDismiss:
			var pending int64
			if err := tx.Model(&model.PostReport{}).
				Where("post_id = ? AND status = ?", postID, model.PostReportStatusPending).
				Count(&pending).Error; err != nil {
				return fmt.Errorf("failed to count pending reports: %w", err)
			}
			if pending == 0 {
				return apperrors.ErrNoPendingReports
			}
		default:
			return fmt.Errorf("unknown moderation action %q", action)
		}

		record := &model.PostModerationAction{
			PostID:       &postID,
			PostAuthorID: post.CreatedBy,
			AdminID:      &adminID,
			Action:       action,
			Note:         note,
		}
		if err := tx.Create(record).Error; err != nil {
			return fmt.Errorf("failed to record moderation action: %w", err)
		}

		switch action {
		case model.ModerationActionHide:
			// UpdateColumn skips updated_at so hiding does not reorder the post
			// in "recently updated" listings.
			if err := tx.Model(&model.Post{}).Where("id = ?", postID).
				UpdateColumn("hidden_at", gorm.Expr("NOW()")).Error; err != nil {
				return fmt.Errorf("failed to hide post: %w", err)
			}
			return closePending(tx, postID, model.PostReportStatusResolved, record.ID)
		case model.ModerationActionUnhide:
			if err := tx.Model(&model.Post{}).Where("id = ?", postID).
				UpdateColumn("hidden_at", nil).Error; err != nil {
				return fmt.Errorf("failed to unhide post: %w", err)
			}
			return nil
		default: // dismiss
			return closePending(tx, postID, model.PostReportStatusDismissed, record.ID)
		}
	})
}

func closePending(tx *gorm.DB, postID, status, actionID string) error {
	err := tx.Model(&model.PostReport{}).
		Where("post_id = ? AND status = ?", postID, model.PostReportStatusPending).
		Updates(map[string]any{"status": status, "action_id": actionID}).Error
	if err != nil {
		return fmt.Errorf("failed to close pending reports: %w", err)
	}
	return nil
}

func (r *postReportRepository) ListActionsByPost(ctx context.Context, postID string, limit, offset int) ([]*model.PostModerationAction, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.PostModerationAction{}).
		Where("post_id = ?", postID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count moderation actions: %w", err)
	}

	var actions []*model.PostModerationAction
	if err := r.db.WithContext(ctx).
		Preload("Admin", preloadUserBrief).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&actions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list moderation actions: %w", err)
	}
	return actions, total, nil
}
