package validator

import (
	"errors"
	"strings"
	"testing"
)

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"valid v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid lowercase v7", "018f4d39-3a4f-7c4f-9b2a-2cf6f8c4f4d3", true},
		{"uppercase rejected", "550E8400-E29B-41D4-A716-446655440000", false},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false},
		{"too short", "550e8400-e29b-41d4-a716-44665544000", false},
		{"non-hex", "550e8400-e29b-41d4-a716-44665544000g", false},
		{"random text", "not-a-uuid", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUUID(tt.in); got != tt.want {
				t.Fatalf("IsValidUUID(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidatePagination(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		offset  int
		wantErr bool
	}{
		{"valid", 10, 0, false},
		{"valid offset", 50, 100, false},
		{"max limit", 100, 0, false},
		{"zero limit", 0, 0, true},
		{"negative limit", -1, 0, true},
		{"limit too big", 101, 0, true},
		{"negative offset", 10, -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePagination(tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePagination(%d,%d) err = %v, wantErr %v", tt.limit, tt.offset, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePostLikeInput(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		postID  string
		userID  string
		wantErr bool
	}{
		{"both valid", validUUID, validUUID, false},
		{"invalid post id", "bad", validUUID, true},
		{"invalid user id", validUUID, "bad", true},
		{"both empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostLikeInput(tt.postID, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePostLikeInput err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePaginationWithDefaults(t *testing.T) {
	tests := []struct {
		name       string
		limit      string
		offset     string
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{"both empty -> defaults", "", "", 10, 0, false},
		{"valid both", "20", "5", 20, 5, false},
		{"non-numeric limit", "abc", "0", 0, 0, true},
		{"non-numeric offset", "10", "xyz", 0, 0, true},
		{"limit exceeds max", "101", "0", 0, 0, true},
		{"negative offset", "10", "-1", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset, err := ValidatePaginationWithDefaults(tt.limit, tt.offset)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if limit != tt.wantLimit || offset != tt.wantOffset {
				t.Fatalf("got (%d,%d) want (%d,%d)", limit, offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"escapes script tag", `hello<script>alert(1)</script>world`, "hello&lt;script&gt;alert(1)&lt;/script&gt;world"},
		{"escapes img onerror", `<img onerror="alert(1)">`, "&lt;img onerror=&#34;alert(1)&#34;&gt;"},
		{"escapes svg onload", `<svg onload="alert(1)">`, "&lt;svg onload=&#34;alert(1)&#34;&gt;"},
		{"escapes ampersand", `foo & bar`, "foo &amp; bar"},
		{"escapes quotes", `he said "hello"`, "he said &#34;hello&#34;"},
		{"escapes single quote", `it's fine`, "it&#39;s fine"},
		{"trims whitespace", "   spaced   ", "spaced"},
		{"empty stays empty", "", ""},
		{"plain text untouched", "just text", "just text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeString(tt.in); got != tt.want {
				t.Fatalf("SanitizeString(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

type sampleStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18,max=120"`
	Role  string `validate:"oneof=admin user guest"`
}

func TestCustomValidator_Validate(t *testing.T) {
	v := NewValidator()

	t.Run("all valid", func(t *testing.T) {
		err := v.Validate(&sampleStruct{
			Name:  "Alice",
			Email: "alice@example.com",
			Age:   30,
			Role:  "admin",
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("collects all errors", func(t *testing.T) {
		err := v.Validate(&sampleStruct{Email: "bad", Age: 5, Role: "other"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
		if len(ve.Errors) == 0 {
			t.Fatal("expected at least one validation error")
		}
		// Build a set of failing fields to make assertions order-independent.
		failing := map[string]string{}
		for _, e := range ve.Errors {
			failing[e.Field] = e.Tag
		}
		if _, ok := failing["Name"]; !ok {
			t.Errorf("expected Name to fail (required), got %v", failing)
		}
		if tag := failing["Email"]; tag != "email" {
			t.Errorf("expected Email tag=email, got %q", tag)
		}
		if tag := failing["Age"]; tag != "min" {
			t.Errorf("expected Age tag=min, got %q", tag)
		}
		if tag := failing["Role"]; tag != "oneof" {
			t.Errorf("expected Role tag=oneof, got %q", tag)
		}

		// Error() should mention the first failure's message in human-readable form.
		if !strings.Contains(ve.Error(), " ") {
			t.Errorf("expected Error() to be a human-readable message, got %q", ve.Error())
		}
	})

	t.Run("empty errors message", func(t *testing.T) {
		ve := ValidationErrors{}
		if msg := ve.Error(); msg != "validation failed" {
			t.Fatalf("got %q, want %q", msg, "validation failed")
		}
	})
}

func TestIsFreeOpenRouterModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{"openrouter free router", "openrouter/free", true},
		{"free suffix", "deepseek/deepseek-v4-flash:free", true},
		{"trimmed free suffix", "  stepfun/step-3.5-flash:free  ", true},
		{"paid model", "openai/gpt-4o", false},
		{"missing free suffix", "stepfun/step-3.5-flash", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFreeOpenRouterModel(tt.model); got != tt.want {
				t.Fatalf("IsFreeOpenRouterModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

type chatMessageModelSample struct {
	Model *string `validate:"omitempty,max=100,free_model"`
}

func TestFreeModelValidation(t *testing.T) {
	v := NewValidator()
	free := "openrouter/free"
	paid := "openai/gpt-4o"

	if err := v.Validate(&chatMessageModelSample{}); err != nil {
		t.Fatalf("expected nil for omitted model, got %v", err)
	}
	if err := v.Validate(&chatMessageModelSample{Model: &free}); err != nil {
		t.Fatalf("expected nil for free model, got %v", err)
	}
	if err := v.Validate(&chatMessageModelSample{Model: &paid}); err == nil {
		t.Fatal("expected validation error for paid model")
	}
}

func BenchmarkIsValidUUID(b *testing.B) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = IsValidUUID(uuid)
	}
}
