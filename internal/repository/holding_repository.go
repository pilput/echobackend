package repository

import (
	"context"
	"errors"
	"fmt"

	apperrors "echobackend/internal/apperror"
	"echobackend/internal/model"

	"gorm.io/gorm"
)

type HoldingRepository interface {
	FindAll(ctx context.Context, userID string, filter *struct {
		Month     *int
		Year      *int
		SortBy    string
		SortOrder string
	}) ([]model.Holding, error)
	FindByID(ctx context.Context, id int64, userID string) (*model.Holding, error)
	Create(ctx context.Context, holding *model.Holding) error
	Update(ctx context.Context, holding *model.Holding) error
	Delete(ctx context.Context, id int64, userID string) error
	FindHoldingTypes(ctx context.Context) ([]model.HoldingType, error)
	FindHoldingTypeByID(ctx context.Context, id int) (*model.HoldingType, error)
	FindForSync(ctx context.Context, userID string, month, year int) ([]model.Holding, error)
	UpdateFields(ctx context.Context, id int64, userID string, fields map[string]any) error
	FindForDuplicate(ctx context.Context, userID string, month, year int) ([]model.Holding, error)
	DeleteByUserMonthYear(ctx context.Context, userID string, month, year int) error
	CountByUserMonthYear(ctx context.Context, userID string, month, year int) (int64, error)
	GetSummary(ctx context.Context, userID string, month, year *int) (invested, currentValue float64, count int64, err error)
	GetTypeBreakdown(ctx context.Context, userID string, month, year *int) ([]struct {
		Name         string
		Invested     float64
		CurrentValue float64
	}, error)
	GetPlatformBreakdown(ctx context.Context, userID string, month, year *int) ([]struct {
		Name         string
		Invested     float64
		CurrentValue float64
	}, error)
	GetTrends(ctx context.Context, userID string, years []int) ([]struct {
		Month        int
		Year         int
		Invested     float64
		CurrentValue float64
	}, error)
	GetMonthlyData(ctx context.Context, userID string, startMonth, startYear, endMonth, endYear int) ([]struct {
		Month        int
		Year         int
		Invested     float64
		CurrentValue float64
		Count        int64
	}, error)
	// FindStockSymbols returns distinct non-empty symbols from holdings whose
	// holding_type.code = 'stock'. Used by the corporate-actions calendar.
	FindStockSymbols(ctx context.Context, userID string) ([]string, error)
}

type holdingRepository struct {
	db *gorm.DB
}

func NewHoldingRepository(db *gorm.DB) HoldingRepository {
	return &holdingRepository{db: db}
}

func (r *holdingRepository) FindAll(ctx context.Context, userID string, filter *struct {
	Month     *int
	Year      *int
	SortBy    string
	SortOrder string
}) ([]model.Holding, error) {
	var holdings []model.Holding
	q := r.db.WithContext(ctx).Preload("HoldingType").Where("user_id = ?", userID)

	if filter != nil {
		if filter.Month != nil {
			q = q.Where("month = ?", *filter.Month)
		}
		if filter.Year != nil {
			q = q.Where("year = ?", *filter.Year)
		}

		sortBy := "created_at"
		switch filter.SortBy {
		case "updated_at", "name", "platform", "invested_amount", "current_value", "holding_type":
			sortBy = filter.SortBy
		}

		sortOrder := "DESC"
		if filter.SortOrder == "asc" {
			sortOrder = "ASC"
		}

		if sortBy == "holding_type" {
			q = q.Order("holding_type_id " + sortOrder + ", created_at " + sortOrder)
		} else {
			q = q.Order(sortBy + " " + sortOrder + ", id " + sortOrder)
		}
	} else {
		q = q.Order("created_at DESC, id DESC")
	}

	if err := q.Find(&holdings).Error; err != nil {
		return nil, fmt.Errorf("failed to find holdings: %w", err)
	}
	return holdings, nil
}

func (r *holdingRepository) FindByID(ctx context.Context, id int64, userID string) (*model.Holding, error) {
	var holding model.Holding
	err := r.db.WithContext(ctx).Preload("HoldingType").Where("id = ? AND user_id = ?", id, userID).First(&holding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrHoldingNotFound
		}
		return nil, fmt.Errorf("failed to find holding: %w", err)
	}
	return &holding, nil
}

func (r *holdingRepository) Create(ctx context.Context, holding *model.Holding) error {
	if err := r.db.WithContext(ctx).Create(holding).Error; err != nil {
		return fmt.Errorf("failed to create holding: %w", err)
	}
	return nil
}

func (r *holdingRepository) Update(ctx context.Context, holding *model.Holding) error {
	result := r.db.WithContext(ctx).Model(&model.Holding{}).
		Where("id = ? AND user_id = ?", holding.ID, holding.UserID).
		Select("Name", "Symbol", "Platform", "HoldingTypeID", "Currency", "InvestedAmount", "CurrentValue", "Units", "AvgBuyPrice", "CurrentPrice", "Notes", "Month", "Year", "LastUpdated", "UpdatedAt").
		Updates(holding)
	if result.Error != nil {
		return fmt.Errorf("failed to update holding: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrHoldingNotFound
	}
	return nil
}

func (r *holdingRepository) Delete(ctx context.Context, id int64, userID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&model.Holding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete holding: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrHoldingNotFound
	}
	return nil
}

func (r *holdingRepository) FindHoldingTypes(ctx context.Context) ([]model.HoldingType, error) {
	var types []model.HoldingType
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&types).Error; err != nil {
		return nil, fmt.Errorf("failed to find holding types: %w", err)
	}
	return types, nil
}

func (r *holdingRepository) FindHoldingTypeByID(ctx context.Context, id int) (*model.HoldingType, error) {
	var ht model.HoldingType
	err := r.db.WithContext(ctx).First(&ht, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrHoldingTypeNotFound
		}
		return nil, fmt.Errorf("failed to find holding type: %w", err)
	}
	return &ht, nil
}

func (r *holdingRepository) FindForSync(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
	var holdings []model.Holding
	err := r.db.WithContext(ctx).
		Where(
			"user_id = ? AND month = ? AND year = ? AND symbol IS NOT NULL AND symbol != '' AND units IS NOT NULL AND units > 0",
			userID, month, year,
		).
		Find(&holdings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find holdings for sync: %w", err)
	}
	return holdings, nil
}

func (r *holdingRepository) UpdateFields(ctx context.Context, id int64, userID string, fields map[string]any) error {
	result := r.db.WithContext(ctx).Model(&model.Holding{}).Where("id = ? AND user_id = ?", id, userID).Updates(fields)
	if result.Error != nil {
		return fmt.Errorf("failed to update holding fields: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrHoldingNotFound
	}
	return nil
}

func (r *holdingRepository) FindForDuplicate(ctx context.Context, userID string, month, year int) ([]model.Holding, error) {
	var holdings []model.Holding
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND month = ? AND year = ?", userID, month, year).
		Find(&holdings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find holdings for duplicate: %w", err)
	}
	return holdings, nil
}

func (r *holdingRepository) DeleteByUserMonthYear(ctx context.Context, userID string, month, year int) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND month = ? AND year = ?", userID, month, year).
		Delete(&model.Holding{}).Error
}

func (r *holdingRepository) CountByUserMonthYear(ctx context.Context, userID string, month, year int) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Holding{}).
		Where("user_id = ? AND month = ? AND year = ?", userID, month, year).
		Count(&count).Error
	return count, err
}

func (r *holdingRepository) GetSummary(ctx context.Context, userID string, month, year *int) (invested, currentValue float64, count int64, err error) {
	q := r.db.WithContext(ctx).Model(&model.Holding{}).Where("user_id = ?", userID)
	if month != nil {
		q = q.Where("month = ?", *month)
	}
	if year != nil {
		q = q.Where("year = ?", *year)
	}

	type result struct {
		Invested     float64
		CurrentValue float64
		Count        int64
	}
	var r1 result
	if err := q.Select("COALESCE(SUM(invested_amount), 0) as invested, COALESCE(SUM(current_value), 0) as current_value, COUNT(*) as count").Scan(&r1).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get summary: %w", err)
	}
	return r1.Invested, r1.CurrentValue, r1.Count, nil
}

func (r *holdingRepository) GetTypeBreakdown(ctx context.Context, userID string, month, year *int) ([]struct {
	Name         string
	Invested     float64
	CurrentValue float64
}, error) {
	q := r.db.WithContext(ctx).Table("holdings h").
		Select("ht.name as name, COALESCE(SUM(h.invested_amount), 0) as invested, COALESCE(SUM(h.current_value), 0) as current_value").
		Joins("JOIN holding_types ht ON ht.id = h.holding_type_id").
		Where("h.user_id = ?", userID)
	if month != nil {
		q = q.Where("h.month = ?", *month)
	}
	if year != nil {
		q = q.Where("h.year = ?", *year)
	}

	var results []struct {
		Name         string
		Invested     float64
		CurrentValue float64
	}
	if err := q.Group("ht.name").Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get type breakdown: %w", err)
	}
	return results, nil
}

func (r *holdingRepository) GetPlatformBreakdown(ctx context.Context, userID string, month, year *int) ([]struct {
	Name         string
	Invested     float64
	CurrentValue float64
}, error) {
	q := r.db.WithContext(ctx).Table("holdings h").
		Select("h.platform as name, COALESCE(SUM(h.invested_amount), 0) as invested, COALESCE(SUM(h.current_value), 0) as current_value").
		Where("h.user_id = ?", userID)
	if month != nil {
		q = q.Where("h.month = ?", *month)
	}
	if year != nil {
		q = q.Where("h.year = ?", *year)
	}

	var results []struct {
		Name         string
		Invested     float64
		CurrentValue float64
	}
	if err := q.Group("h.platform").Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get platform breakdown: %w", err)
	}
	return results, nil
}

func (r *holdingRepository) GetTrends(ctx context.Context, userID string, years []int) ([]struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
}, error) {
	q := r.db.WithContext(ctx).Table("holdings").
		Select("month, year, COALESCE(SUM(invested_amount), 0) as invested, COALESCE(SUM(current_value), 0) as current_value").
		Where("user_id = ?", userID)
	if len(years) > 0 {
		q = q.Where("year IN ?", years)
	}

	var results []struct {
		Month        int
		Year         int
		Invested     float64
		CurrentValue float64
	}
	if err := q.Group("year, month").Order("year ASC, month ASC").Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get trends: %w", err)
	}
	return results, nil
}

func (r *holdingRepository) GetMonthlyData(ctx context.Context, userID string, startMonth, startYear, endMonth, endYear int) ([]struct {
	Month        int
	Year         int
	Invested     float64
	CurrentValue float64
	Count        int64
}, error) {
	var results []struct {
		Month        int
		Year         int
		Invested     float64
		CurrentValue float64
		Count        int64
	}

	err := r.db.WithContext(ctx).Table("holdings").
		Select("month, year, COALESCE(SUM(invested_amount), 0) as invested, COALESCE(SUM(current_value), 0) as current_value, COUNT(*) as count").
		Where("user_id = ? AND ((year > ? OR (year = ? AND month >= ?)) AND (year < ? OR (year = ? AND month <= ?)))", userID, endYear, endYear, endMonth, startYear, startYear, startMonth).
		Group("year, month").
		Order("year ASC, month ASC").
		Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly data: %w", err)
	}
	return results, nil
}

func (r *holdingRepository) FindStockSymbols(ctx context.Context, userID string) ([]string, error) {
	var symbols []string
	err := r.db.WithContext(ctx).
		Table("holdings h").
		Select("DISTINCT h.symbol").
		Joins("JOIN holding_types ht ON ht.id = h.holding_type_id").
		Where("h.user_id = ? AND LOWER(ht.code) = 'stock' AND h.symbol IS NOT NULL AND h.symbol != ''", userID).
		Pluck("h.symbol", &symbols).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find stock symbols: %w", err)
	}
	return symbols, nil
}
