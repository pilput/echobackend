package repository

import (
	"context"
	"strings"
	"time"

	"echobackend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const corporateActionDedupeDateFormat = "2006-01-02"

// CorporateActionRepository persists the IDX dividend & RUPS calendar in Postgres.
type CorporateActionRepository interface {
	// ExistsInRange reports whether any corporate action rows already fall
	// within [from, to]. Used to decide whether an IDX fetch is needed.
	ExistsInRange(ctx context.Context, from, to time.Time) (bool, error)
	FindByDateRange(ctx context.Context, from, to time.Time) ([]model.CorporateAction, error)
	// UpsertMany inserts actions, updating mutable fields on conflict of
	// (symbol, type, event_date).
	UpsertMany(ctx context.Context, actions []model.CorporateAction) error
}

type corporateActionRepository struct {
	db *gorm.DB
}

func NewCorporateActionRepository(db *gorm.DB) CorporateActionRepository {
	return &corporateActionRepository{db: db}
}

func (r *corporateActionRepository) ExistsInRange(ctx context.Context, from, to time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.CorporateAction{}).
		Where("event_date BETWEEN ? AND ?", from, to).
		Count(&count).Error
	return count > 0, err
}

func (r *corporateActionRepository) FindByDateRange(ctx context.Context, from, to time.Time) ([]model.CorporateAction, error) {
	var actions []model.CorporateAction
	err := r.db.WithContext(ctx).
		Where("event_date BETWEEN ? AND ?", from, to).
		Order("event_date ASC").
		Find(&actions).Error
	return actions, err
}

func (r *corporateActionRepository) UpsertMany(ctx context.Context, actions []model.CorporateAction) error {
	actions = dedupeCorporateActions(actions)
	if len(actions) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "symbol"}, {Name: "type"}, {Name: "event_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "cum_date", "rec_date", "pay_date", "amount", "currency", "note", "market", "updated_at"}),
	}).Create(&actions).Error
}

// dedupeCorporateActions collapses rows sharing the same (symbol, type,
// event_date) — the unique constraint target — into one. IDX occasionally
// reports the same company/date more than once (e.g. multiple RUPS agenda
// items), and a single INSERT ... ON CONFLICT DO UPDATE statement errors
// ("cannot affect row a second time") if its target key repeats within the
// batch, so duplicates must be merged before reaching the DB. Distinct notes
// are concatenated so no agenda information is silently dropped.
func dedupeCorporateActions(actions []model.CorporateAction) []model.CorporateAction {
	type key struct {
		symbol string
		typ    string
		date   string
	}
	order := make([]key, 0, len(actions))
	merged := make(map[key]model.CorporateAction, len(actions))

	for _, a := range actions {
		k := key{a.Symbol, a.Type, a.EventDate.Format(corporateActionDedupeDateFormat)}
		existing, ok := merged[k]
		if !ok {
			merged[k] = a
			order = append(order, k)
			continue
		}
		if existing.Note != a.Note && a.Note != "" && !strings.Contains(existing.Note, a.Note) {
			if existing.Note == "" {
				existing.Note = a.Note
			} else {
				existing.Note = existing.Note + "; " + a.Note
			}
		}
		if existing.Amount == nil {
			existing.Amount = a.Amount
		}
		if existing.CumDate == nil {
			existing.CumDate = a.CumDate
		}
		if existing.RecDate == nil {
			existing.RecDate = a.RecDate
		}
		if existing.PayDate == nil {
			existing.PayDate = a.PayDate
		}
		merged[k] = existing
	}

	out := make([]model.CorporateAction, 0, len(order))
	for _, k := range order {
		out = append(out, merged[k])
	}
	return out
}
