package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "echobackend/internal/apperror"

	"github.com/labstack/echo/v5"
)

// The frontend reads the rejection reason from errors[].tag == "spam"; this
// pins the envelope it depends on (shared with axumbackend).
func TestRespondError_SpamRejectionEnvelope(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := echo.NewContext(req, rec, echo.New())

	err := apperrors.NewSpamRejection(apperrors.ErrContentSpamRejected, "contains banned keyword \"gacor\"")
	if respErr := respondError(c, "Failed to create comment", err); respErr != nil {
		t.Fatalf("respondError returned %v", respErr)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	var body struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Errors  []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
			Tag     string `json:"tag"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error != "Content rejected by anti-spam policy" {
		t.Errorf("error = %q", body.Error)
	}
	if len(body.Errors) != 1 || body.Errors[0].Tag != "spam" || body.Errors[0].Field != "text" ||
		body.Errors[0].Message != "contains banned keyword \"gacor\"" {
		t.Errorf("errors = %+v, want one spam entry for field text carrying the bare reason", body.Errors)
	}
}
