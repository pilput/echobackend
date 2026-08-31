package service

import (
	"context"
	"sort"
	"time"

	"echobackend/internal/dto"
	"echobackend/internal/model"
	"echobackend/internal/repository"
	"echobackend/pkg/market"
)

const calendarDateFormat = "2006-01-02"

// CorporateActionService fetches dividend and RUPS events for the user's
// stock holdings and persists them in Postgres.
type CorporateActionService interface {
	GetCalendar(ctx context.Context, userID string, year, month int) (*dto.CorporateActionCalendarResponse, error)
}

type corporateActionService struct {
	idxClient market.CorporateActionClient // RapidAPI IDX (IDX stocks)
	repo      repository.CorporateActionRepository
}

// NewCorporateActionService creates the service.
func NewCorporateActionService(
	idxClient market.CorporateActionClient,
	repo repository.CorporateActionRepository,
) CorporateActionService {
	return &corporateActionService{
		idxClient: idxClient,
		repo:      repo,
	}
}

// GetCalendar returns corporate actions (dividend + RUPS) for the given month.
//
// Results are persisted in Postgres. If the month already has stored rows,
// they are served directly without calling IDX again. Otherwise IDX is
// queried, the results are upserted, and the freshly stored rows are
// returned. Individual API errors are swallowed (fail-open) so that a
// partial result is always returned.
func (s *corporateActionService) GetCalendar(ctx context.Context, _ string, year, month int) (*dto.CorporateActionCalendarResponse, error) {
	if month < 1 || month > 12 {
		month = int(time.Now().Month())
	}

	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, -1)

	cached := false
	if s.repo != nil {
		exists, err := s.repo.ExistsInRange(ctx, from, to)
		if err == nil {
			cached = exists
		}
	}

	if !cached && s.idxClient != nil {
		idxActions, _ := s.idxClient.GetCorporateActions(ctx, from, to)
		if len(idxActions) > 0 && s.repo != nil {
			rows := make([]model.CorporateAction, 0, len(idxActions))
			for _, a := range idxActions {
				rows = append(rows, model.CorporateAction{
					Symbol:    a.Symbol,
					Name:      a.Name,
					Type:      string(a.Type),
					EventDate: a.Date,
					CumDate:   a.CumDate,
					RecDate:   a.RecDate,
					PayDate:   a.PayDate,
					Amount:    a.Amount,
					Currency:  a.Currency,
					Note:      a.Note,
					Market:    a.Market,
				})
			}
			_ = s.repo.UpsertMany(ctx, rows)
		}
	}

	var stored []model.CorporateAction
	if s.repo != nil {
		stored, _ = s.repo.FindByDateRange(ctx, from, to)
	}

	sort.Slice(stored, func(i, j int) bool {
		return stored[i].EventDate.Before(stored[j].EventDate)
	})

	responses := make([]dto.CorporateActionResponse, 0, len(stored))
	for _, a := range stored {
		r := dto.CorporateActionResponse{
			Symbol:   a.Symbol,
			Name:     a.Name,
			Type:     a.Type,
			Date:     a.EventDate.Format(calendarDateFormat),
			Currency: a.Currency,
			Note:     a.Note,
			Market:   a.Market,
			Amount:   a.Amount,
		}
		if a.CumDate != nil {
			cs := a.CumDate.Format(calendarDateFormat)
			r.CumDate = &cs
		}
		if a.RecDate != nil {
			rs := a.RecDate.Format(calendarDateFormat)
			r.RecDate = &rs
		}
		if a.PayDate != nil {
			ps := a.PayDate.Format(calendarDateFormat)
			r.PayDate = &ps
		}
		responses = append(responses, r)
	}

	return &dto.CorporateActionCalendarResponse{
		From:    from.Format(calendarDateFormat),
		To:      to.Format(calendarDateFormat),
		Total:   len(responses),
		Cached:  cached,
		Actions: responses,
	}, nil
}
