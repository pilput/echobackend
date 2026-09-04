package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Default Argon2id parameters aligned with OWASP Password Storage recommendations.
const (
	AlgorithmArgon2id = "argon2id"
	Argon2Version     = argon2.Version // 19 (0x13)
	DefaultMemory     = 64 * 1024      // 64 MB in KiB
	DefaultTime       = 1              // 1 iteration
	DefaultThreads    = 4              // 4 threads
	DefaultKeyLen     = 32             // 32 bytes output key
	DefaultSaltLen    = 16             // 16 bytes random salt
)

var (
	ErrInvalidHash         = errors.New("invalid or malformed password hash")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
	salt    []byte
	key     []byte
}

// Hash creates a cryptographically secure Argon2id hash of a password in standard PHC format:
// $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
func Hash(password string) (string, error) {
	salt := make([]byte, DefaultSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, DefaultTime, DefaultMemory, DefaultThreads, DefaultKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		AlgorithmArgon2id, Argon2Version, DefaultMemory, DefaultTime, DefaultThreads, b64Salt, b64Key), nil
}

// Compare verifies whether the given plaintext password matches the hashedPassword.
// It supports both modern Argon2id hashes and legacy bcrypt hashes ($2a$, $2b$, $2y$).
// Returns (true, nil) on match, (false, nil) on mismatch, and (false, err) on malformed hash.
func Compare(hashedPassword, password string) (bool, error) {
	if isBcrypt(hashedPassword) {
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		if err != nil {
			if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}

	if strings.HasPrefix(hashedPassword, "$argon2id$") {
		params, err := parseArgon2idHash(hashedPassword)
		if err != nil {
			return false, err
		}

		computedKey := argon2.IDKey([]byte(password), params.salt, params.time, params.memory, params.threads, uint32(len(params.key)))
		if subtle.ConstantTimeCompare(params.key, computedKey) == 1 {
			return true, nil
		}
		return false, nil
	}

	return false, ErrInvalidHash
}

// NeedsRehash reports whether the given hashedPassword is using an older algorithm (e.g. bcrypt)
// or different Argon2id parameters than the current system defaults.
// When NeedsRehash returns true after a successful login, the application should re-hash
// the password using Hash() and update the database.
func NeedsRehash(hashedPassword string) bool {
	if isBcrypt(hashedPassword) {
		return true
	}

	params, err := parseArgon2idHash(hashedPassword)
	if err != nil {
		return true
	}

	if params.memory != DefaultMemory ||
		params.time != DefaultTime ||
		params.threads != DefaultThreads ||
		len(params.key) != DefaultKeyLen {
		return true
	}

	return false
}

func isBcrypt(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$")
}

// parseArgon2idHash parses PHC format:
// $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
func parseArgon2idHash(hashedPassword string) (*argon2Params, error) {
	parts := strings.Split(hashedPassword, "$")
	// Expected parts: ["", "argon2id", "v=19", "m=65536,t=1,p=4", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return nil, ErrInvalidHash
	}

	if parts[1] != AlgorithmArgon2id {
		return nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, ErrInvalidHash
	}
	if version != Argon2Version {
		return nil, ErrIncompatibleVersion
	}

	var memory, time uint32
	var threads uint8
	subparts := strings.Split(parts[3], ",")
	if len(subparts) != 3 {
		return nil, ErrInvalidHash
	}

	for _, subpart := range subparts {
		kv := strings.SplitN(subpart, "=", 2)
		if len(kv) != 2 {
			return nil, ErrInvalidHash
		}
		val, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return nil, ErrInvalidHash
		}
		switch kv[0] {
		case "m":
			memory = uint32(val)
		case "t":
			time = uint32(val)
		case "p":
			if val > 255 {
				return nil, ErrInvalidHash
			}
			threads = uint8(val)
		default:
			return nil, ErrInvalidHash
		}
	}

	if memory == 0 || time == 0 || threads == 0 {
		return nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, ErrInvalidHash
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, ErrInvalidHash
	}

	return &argon2Params{
		memory:  memory,
		time:    time,
		threads: threads,
		salt:    salt,
		key:     key,
	}, nil
}
