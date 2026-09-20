// Package uid generates UUIDs in the application, for the rare rows whose id
// has to be known before the INSERT.
//
// Primary keys are normally left to Postgres, whose default is uuidv7()
// (migration 009). This package produces the same version so ids generated on
// either side sort and index alike.
package uid

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"time"
)

// NewV7 returns a RFC 9562 version 7 UUID: a 48-bit big-endian Unix
// millisecond timestamp followed by 74 random bits, which keeps generated ids
// roughly time-ordered and therefore friendly to B-tree inserts.
func NewV7() (string, error) {
	var b [16]byte
	// Bytes 0..5 are overwritten with the timestamp below.
	if _, err := io.ReadFull(rand.Reader, b[6:]); err != nil {
		return "", err
	}

	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)

	b[6] = (b[6] & 0x0f) | 0x70 // version 7
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 9562 variant

	return format(b), nil
}

// format renders the 16 bytes in the canonical 8-4-4-4-12 hyphenated form.
func format(b [16]byte) string {
	var buf [36]byte
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])
	return string(buf[:])
}
