package password

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndCompare_Argon2id(t *testing.T) {
	plainPassword := "SecretP@ssw0rd!2026"

	hashed, err := Hash(plainPassword)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !strings.HasPrefix(hashed, "$argon2id$v=19$") {
		t.Fatalf("expected argon2id prefix, got: %s", hashed)
	}

	// Correct password
	match, err := Compare(hashed, plainPassword)
	if err != nil {
		t.Fatalf("unexpected error comparing correct password: %v", err)
	}
	if !match {
		t.Fatalf("expected password to match")
	}

	// Wrong password
	match, err = Compare(hashed, "WrongP@ssword")
	if err != nil {
		t.Fatalf("unexpected error comparing wrong password: %v", err)
	}
	if match {
		t.Fatalf("expected wrong password to not match")
	}

	// NeedsRehash on current Argon2id should be false
	if NeedsRehash(hashed) {
		t.Fatalf("expected NeedsRehash to be false for freshly generated hash")
	}
}

func TestCompare_LongPassword(t *testing.T) {
	// Passwords > 72 bytes are truncated by bcrypt, but Argon2id handles them without truncation
	longPassword := strings.Repeat("A-very-long-secure-passphrase-with-lots-of-entropy!", 3) // > 150 chars

	hashed, err := Hash(longPassword)
	if err != nil {
		t.Fatalf("failed to hash long password: %v", err)
	}

	match, err := Compare(hashed, longPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !match {
		t.Fatalf("expected long password to match")
	}

	// Altering one character at the very end
	wrongLongPassword := longPassword[:len(longPassword)-1] + "X"
	match, err = Compare(hashed, wrongLongPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match {
		t.Fatalf("expected altered long password to fail")
	}
}

func TestBackwardCompatibility_Bcrypt(t *testing.T) {
	plainPassword := "LegacyBcryptPassword123"

	bcryptHashBytes, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}
	bcryptHash := string(bcryptHashBytes)

	// Verify legacy bcrypt hash matches
	match, err := Compare(bcryptHash, plainPassword)
	if err != nil {
		t.Fatalf("unexpected error verifying bcrypt hash: %v", err)
	}
	if !match {
		t.Fatalf("expected bcrypt hash to match")
	}

	// Verify wrong password fails on bcrypt hash
	match, err = Compare(bcryptHash, "IncorrectPassword")
	if err != nil {
		t.Fatalf("unexpected error verifying wrong password on bcrypt hash: %v", err)
	}
	if match {
		t.Fatalf("expected wrong password to fail on bcrypt hash")
	}

	// NeedsRehash MUST be true for bcrypt hashes to enable automatic upgrade
	if !NeedsRehash(bcryptHash) {
		t.Fatalf("expected NeedsRehash to be true for legacy bcrypt hash")
	}
}

func TestNeedsRehash_DifferentArgon2Params(t *testing.T) {
	// A valid Argon2id hash with lower memory (e.g. 19456 KiB)
	oldParamHash := "$argon2id$v=19$m=19456,t=1,p=1$c29tZXNhbHR2YWx1ZQ$c29tZWtleXZhbHVldGhhdGlzbG9uZw"

	if !NeedsRehash(oldParamHash) {
		t.Fatalf("expected NeedsRehash to be true for Argon2id hash with different parameters")
	}
}

func TestCompare_InvalidOrMalformedHash(t *testing.T) {
	cases := []string{
		"",
		"plain_password_not_hash",
		"$argon2id$v=99$m=65536,t=1,p=4$salt$hash",       // incompatible version
		"$argon2id$v=19$invalid_params$salt$hash",        // invalid params
		"$argon2id$v=19$m=65536,t=1,p=4$bad_b64!!!$hash", // bad base64 salt
		"$md5$123456", // unsupported format
	}

	for _, c := range cases {
		match, err := Compare(c, "anypassword")
		if match {
			t.Errorf("expected match=false for malformed hash %q", c)
		}
		if err == nil {
			t.Errorf("expected error for malformed hash %q", c)
		}
	}
}
