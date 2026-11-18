package blog

import (
	"context"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/pkg/validator"
)

type PostReportService interface {
	ReportPost(ctx context.Context, postID, reporterID string, req CreatePostReportRequest) error
	ListReportedPosts(ctx context.Context, status string, limit, offset int) ([]PostReportGroup, int64, error)
	ListPostReports(ctx context.Context, postID, status string, limit, offset int) ([]*PostReportResponse, int64, error)
	ModeratePost(ctx context.Context, postID, adminID string, req ModeratePostRequest) error
	ListModerationActions(ctx context.Context, postID string, limit, offset int) ([]*ModerationActionResponse, int64, error)
	ReportStats(ctx context.Context) (*ReportStatsResponse, error)
}

type postReportService struct {
	reportRepo PostReportRepository
	postRepo   PostRepository
}

func NewPostReportService(reportRepo PostReportRepository, postRepo PostRepository) PostReportService {
	return &postReportService{reportRepo: reportRepo, postRepo: postRepo}
}

func validReportStatus(status string) bool {
	switch status {
	case PostReportStatusPending, PostReportStatusResolved, PostReportStatusDismissed:
		return true
	}
	return false
}

func (s *postReportService) ReportPost(ctx context.Context, postID, reporterID string, req CreatePostReportRequest) error {
	if !validator.IsValidUUID(postID) {
		return ErrInvalidPostID
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
		return ErrCannotReportOwnPost
	}

	// An empty string carries no information; store it as NULL.
	details := req.Details
	if details != nil && *details == "" {
		details = nil
	}

	return s.reportRepo.Create(ctx, &PostReport{
		PostID:     postID,
		ReporterID: reporterID,
		Reason:     req.Reason,
		Details:    details,
		Status:     PostReportStatusPending,
	})
}

func (s *postReportService) ListReportedPosts(ctx context.Context, status string, limit, offset int) ([]PostReportGroup, int64, error) {
	if status == "" {
		status = PostReportStatusPending
	}
	if !validReportStatus(status) {
		return nil, 0, ErrInvalidReportStatus
	}
	return s.reportRepo.ListGroupedByPost(ctx, status, limit, offset)
}

func (s *postReportService) ListPostReports(ctx context.Context, postID, status string, limit, offset int) ([]*PostReportResponse, int64, error) {
	if !validator.IsValidUUID(postID) {
		return nil, 0, ErrInvalidPostID
	}
	if status != "" && !validReportStatus(status) {
		return nil, 0, ErrInvalidReportStatus
	}
	reports, total, err := s.reportRepo.ListByPost(ctx, postID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*PostReportResponse, 0, len(reports))
	for _, r := range reports {
		out = append(out, PostReportToResponse(r))
	}
	return out, total, nil
}

func (s *postReportService) ModeratePost(ctx context.Context, postID, adminID string, req ModeratePostRequest) error {
	if !validator.IsValidUUID(postID) {
		return ErrInvalidPostID
	}
	if !validator.IsValidUUID(adminID) {
		return apperrors.ErrInvalidUserID
	}
	return s.reportRepo.Moderate(ctx, postID, adminID, req.Action, req.Note)
}

func (s *postReportService) ListModerationActions(ctx context.Context, postID string, limit, offset int) ([]*ModerationActionResponse, int64, error) {
	if !validator.IsValidUUID(postID) {
		return nil, 0, ErrInvalidPostID
	}
	actions, total, err := s.reportRepo.ListActionsByPost(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*ModerationActionResponse, 0, len(actions))
	for _, a := range actions {
		out = append(out, ModerationActionToResponse(a))
	}
	return out, total, nil
}

func (s *postReportService) ReportStats(ctx context.Context) (*ReportStatsResponse, error) {
	return s.reportRepo.Stats(ctx)
}
