package apperror

// SpamRejection carries the user-facing reason an anti-spam rule rejected
// content. It unwraps to Kind (ErrPostSpamRejected or ErrContentSpamRejected),
// so errors.Is keeps working while handlers can still read Reason directly.
type SpamRejection struct {
	Kind   error
	Reason string
}

func NewSpamRejection(kind error, reason string) *SpamRejection {
	return &SpamRejection{Kind: kind, Reason: reason}
}

func (e *SpamRejection) Error() string {
	return e.Kind.Error() + ": " + e.Reason
}

func (e *SpamRejection) Unwrap() error {
	return e.Kind
}
