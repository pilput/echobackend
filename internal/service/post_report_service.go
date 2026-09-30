package service

import (
	"context"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
	"echobackend/pkg/validator"
)

type PostReportService interface {
	ReportPost(ctx context.Context, postID, reporterID string, req dto.CreatePostReportRequest) error
	ListReportedPosts(ctx context.Context, status string, limit, offset int) ([]dto.PostReportGroup, int64, error)
	ListPostReports(ctx context.Context, postID, status string, limit, offset int) ([]*dto.PostReportResponse, int64, error)
	ModeratePost(ctx context.Context, postID, adminID string, req dto.ModeratePostRequest) error
	ListModerationActions(ctx context.Context, postID string, limit, offset int) ([]*dto.ModerationActionResponse, int64, error)
}

type postReportService struct {
	reportRepo repository.PostReportRepository
	postRepo   repository.PostRepository
}

func NewPostReportService(reportRepo repository.PostReportRepository, postRepo repository.PostRepository) PostReportService {
	return &postReportService{reportRepo: reportRepo, postRepo: postRepo}
}

func validReportStatus(status string) bool {
	switch status {
	case model.PostReportStatusPending, model.PostReportStatusResolved, model.PostReportStatusDismissed:
		return true
	}
	return false
}

func (s *postReportService) ReportPost(ctx context.Context, postID, reporterID string, req dto.CreatePostReportRequest) error {
	if !validator.IsValidUUID(postID) {
		return apperrors.ErrInvalidPostID
	}
	if !validator.IsValidUUID(reporterID) {
		return apperrors.ErrInvalidUserID
	}

	post, err := s.postRepo.GetPostByID(ctx, postID)
	if err != nil {
		return fmt.Errorf("failed to check post existence: %w", err)
	}
	if err := ensurePostInteractable(post); err != nil {
		return err
	}
	if post.CreatedBy != nil && *post.CreatedBy == reporterID {
		return apperrors.ErrCannotReportOwnPost
	}

	// An empty string carries no information; store it as NULL.
	details := req.Details
	if details != nil && *details == "" {
		details = nil
	}

	return s.reportRepo.Create(ctx, &model.PostReport{
		PostID:     postID,
		ReporterID: reporterID,
		Reason:     req.Reason,
		Details:    details,
		Status:     model.PostReportStatusPending,
	})
}

func (s *postReportService) ListReportedPosts(ctx context.Context, status string, limit, offset int) ([]dto.PostReportGroup, int64, error) {
	if status == "" {
		status = model.PostReportStatusPending
	}
	if !validReportStatus(status) {
		return nil, 0, apperrors.ErrInvalidReportStatus
	}
	return s.reportRepo.ListGroupedByPost(ctx, status, limit, offset)
}

func (s *postReportService) ListPostReports(ctx context.Context, postID, status string, limit, offset int) ([]*dto.PostReportResponse, int64, error) {
	if !validator.IsValidUUID(postID) {
		return nil, 0, apperrors.ErrInvalidPostID
	}
	if status != "" && !validReportStatus(status) {
		return nil, 0, apperrors.ErrInvalidReportStatus
	}
	reports, total, err := s.reportRepo.ListByPost(ctx, postID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*dto.PostReportResponse, 0, len(reports))
	for _, r := range reports {
		out = append(out, dto.PostReportToResponse(r))
	}
	return out, total, nil
}

func (s *postReportService) ModeratePost(ctx context.Context, postID, adminID string, req dto.ModeratePostRequest) error {
	if !validator.IsValidUUID(postID) {
		return apperrors.ErrInvalidPostID
	}
	if !validator.IsValidUUID(adminID) {
		return apperrors.ErrInvalidUserID
	}
	return s.reportRepo.Moderate(ctx, postID, adminID, req.Action, req.Note)
}

func (s *postReportService) ListModerationActions(ctx context.Context, postID string, limit, offset int) ([]*dto.ModerationActionResponse, int64, error) {
	if !validator.IsValidUUID(postID) {
		return nil, 0, apperrors.ErrInvalidPostID
	}
	actions, total, err := s.reportRepo.ListActionsByPost(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*dto.ModerationActionResponse, 0, len(actions))
	for _, a := range actions {
		out = append(out, dto.ModerationActionToResponse(a))
	}
	return out, total, nil
}
