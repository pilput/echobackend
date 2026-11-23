package storage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/gen2brain/webp"
)

// noisyPNG encodes a noisy image so encoders cannot shrink it to nothing.
func noisyPNG(t *testing.T, w, h int, alpha bool) []byte {
	t.Helper()
	rng := rand.New(rand.NewPCG(1, 2))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if alpha {
				a = uint8(x % 256)
			}
			v := rng.Uint32()
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(v), G: uint8(v >> 8), B: uint8(v >> 16), A: a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestOptimizeImageDownscalesToWebP(t *testing.T) {
	in := noisyPNG(t, 2400, 1200, false)
	out, ct, ext := OptimizeImage(in, "image/png", ".png", 1600)
	if ct != "image/webp" || ext != ".webp" {
		t.Fatalf("got %s %s, want image/webp .webp", ct, ext)
	}
	if len(out) >= len(in) {
		t.Fatalf("not smaller: %d >= %d", len(out), len(in))
	}
	img, err := webp.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 1600 || b.Dy() != 800 {
		t.Fatalf("got %dx%d, want 1600x800", b.Dx(), b.Dy())
	}
}

func TestOptimizeImageKeepsAlpha(t *testing.T) {
	in := noisyPNG(t, 800, 800, true)
	out, ct, _ := OptimizeImage(in, "image/png", ".png", 1600)
	if ct != "image/webp" {
		t.Fatalf("got %s, want image/webp", ct)
	}
	img, err := webp.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a == 0xffff {
		t.Fatal("alpha was dropped")
	}
}

func TestOptimizeImagePassesThroughGIFAndGarbage(t *testing.T) {
	for _, tc := range []struct{ data, ct, ext string }{
		{"GIF89a\x01\x00", "image/gif", ".gif"},
		{"\x89PNG\r\n\x1a\nnot really", "image/png", ".png"},
	} {
		out, ct, ext := OptimizeImage([]byte(tc.data), tc.ct, tc.ext, 1600)
		if string(out) != tc.data || ct != tc.ct || ext != tc.ext {
			t.Fatalf("%s: expected original back, got %s %s", tc.ct, ct, ext)
		}
	}
}
