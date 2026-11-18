package guilds

import (
	"context"
	"errors"
	"testing"

	apperrors "echobackend/internal/apperror"
)

const validUserID = "018f4d39-3a4f-7c4f-9b2a-2cf6f8c4f4d3"

// The guild repo mock panics on any unstubbed call, so this also proves the
// guard runs before any database access.
func TestGuildService_RejectsSpam(t *testing.T) {
	svc := NewGuildService(&mockGuildRepo{})

	description := "Komunitas slot gacor, daftar di https://slot88.example"
	createReq := &CreateGuildRequest{Name: "Info Harian", Description: &description}
	if _, err := svc.CreateGuild(context.Background(), validUserID, createReq); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("CreateGuild err = %v, want ErrContentSpamRejected", err)
	}

	name := "T O G E L Club"
	updateReq := &UpdateGuildRequest{Name: &name}
	if _, err := svc.UpdateGuild(context.Background(), "info-harian", validUserID, updateReq); !errors.Is(err, apperrors.ErrContentSpamRejected) {
		t.Fatalf("UpdateGuild err = %v, want ErrContentSpamRejected", err)
	}
}
