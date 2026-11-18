package portfolio

import "errors"

// Portfolio domain errors; handlers map them to HTTP statuses in
// respondHoldingError.
var (
	ErrHoldingNotFound       = errors.New("holding not found")
	ErrHoldingTypeNotFound   = errors.New("holding type not found")
	ErrHoldingNotOwned       = errors.New("not authorized to modify this holding")
	ErrHoldingDuplicateSame  = errors.New("source and target month/year are the same")
	ErrHoldingInvalidRange   = errors.New("end month/year must be on or before start month/year")
	ErrHoldingRangeTooLarge  = errors.New("requested month/year range is too large")
	ErrHoldingInvalidDecimal = errors.New("invalid decimal value")
)

// HoldingInvalidDecimalError reports a holding amount field (invested_amount,
// current_value, units, avg_buy_price, current_price) that is not a plain
// decimal number. Field is the JSON name; errors.Is matches
// ErrHoldingInvalidDecimal.
type HoldingInvalidDecimalError struct {
	Field string
}

func (e *HoldingInvalidDecimalError) Error() string {
	return "Invalid decimal value for " + e.Field
}

func (e *HoldingInvalidDecimalError) Is(target error) bool {
	return target == ErrHoldingInvalidDecimal
}
