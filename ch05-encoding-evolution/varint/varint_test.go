package varint

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"math/rand/v2"
	"testing"
)

// Checkbox 1: known byte patterns.
func TestKnownVectors(t *testing.T) {
	cases := []struct {
		in   uint64
		want []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{127, []byte{0x7f}},
		{128, []byte{0x80, 0x01}},
		{300, []byte{0xac, 0x02}},
		{1001, []byte{0xe9, 0x07}},
		{math.MaxUint64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	}
	for _, c := range cases {
		got := AppendUvarint(nil, c.in)
		if !bytes.Equal(got, c.want) {
			t.Errorf("AppendUvarint(%d) = % x, want % x", c.in, got, c.want)
		}
		x, n, err := ReadUvarint(got)
		if err != nil || x != c.in || n != len(got) {
			t.Errorf("ReadUvarint(% x) = %d, %d, %v", got, x, n, err)
		}
	}
}

// Checkbox 2: ZigZag mapping.
func TestZigZag(t *testing.T) {
	cases := []struct {
		in   int64
		want uint64
	}{
		{0, 0},
		{-1, 1},
		{1, 2},
		{-2, 3},
		{2, 4},
		{250, 500},
		{-250, 499},
		{math.MaxInt64, math.MaxUint64 - 1},
		{math.MinInt64, math.MaxUint64},
	}
	for _, c := range cases {
		if got := ZigZag(c.in); got != c.want {
			t.Errorf("ZigZag(%d) = %d, want %d", c.in, got, c.want)
		}
		if got := UnZigZag(c.want); got != c.in {
			t.Errorf("UnZigZag(%d) = %d, want %d", c.want, got, c.in)
		}
	}
}

// randomValue spreads values across all byte lengths. A plain
// rand.Uint64() is almost always a 10-byte varint, which would never
// exercise the short paths.
func randomValue(r *rand.Rand) uint64 {
	bits := r.IntN(65) // 0..64 significant bits
	if bits == 0 {
		return 0
	}
	return r.Uint64() >> (64 - bits)
}

// Checkbox 3: byte-for-byte agreement with the standard library.
func TestMatchesStdlib(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	const N = 1_000_000
	for i := 0; i < N; i++ {
		u := randomValue(r)
		if got, want := AppendUvarint(nil, u), binary.AppendUvarint(nil, u); !bytes.Equal(got, want) {
			t.Fatalf("unsigned %d: got % x, want % x", u, got, want)
		}

		s := int64(randomValue(r))
		if got, want := AppendVarint(nil, s), binary.AppendVarint(nil, s); !bytes.Equal(got, want) {
			t.Fatalf("signed %d: got % x, want % x", s, got, want)
		}
	}
}

// Checkbox 5: truncated / malformed input returns errors, never panics.
func TestMalformed(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want error
	}{
		{"empty", nil, ErrTruncated},
		{"single continuation byte", []byte{0x80}, ErrTruncated},
		{"300 missing last byte", []byte{0xac}, ErrTruncated},
		{"nine continuation bytes", bytes.Repeat([]byte{0xff}, 9), ErrTruncated},
		{"eleven bytes", append(bytes.Repeat([]byte{0x80}, 10), 0x01), ErrOverflow},
		{"10th byte too big", append(bytes.Repeat([]byte{0xff}, 9), 0x02), ErrOverflow},
	}
	for _, c := range cases {
		_, _, err := ReadUvarint(c.in)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

// Checkbox 4: go test -fuzz=FuzzRoundTrip -fuzztime=60s
func FuzzRoundTrip(f *testing.F) {
	for _, seed := range []uint64{0, 1, 127, 128, 300, 1001, math.MaxUint64} {
		f.Add(seed, int64(seed))
	}
	f.Fuzz(func(t *testing.T, u uint64, s int64) {
		enc := AppendUvarint(nil, u)
		got, n, err := ReadUvarint(enc)
		if err != nil || got != u || n != len(enc) {
			t.Fatalf("uvarint round trip %d: got %d, n=%d, err=%v", u, got, n, err)
		}
		if gs := UnZigZag(ZigZag(s)); gs != s {
			t.Fatalf("zigzag round trip %d: got %d", s, gs)
		}
	})
}

// Bonus fuzz: arbitrary garbage bytes must never panic, and whatever
// we accept must agree with the standard library decoder.
func FuzzDecodeGarbage(f *testing.F) {
	f.Add([]byte{0xac, 0x02})
	f.Add([]byte{0x80})
	f.Fuzz(func(t *testing.T, b []byte) {
		x, n, err := ReadUvarint(b)
		sx, sn := binary.Uvarint(b)
		if err == nil {
			if sn != n || sx != x {
				t.Fatalf("% x: ours (%d,%d), stdlib (%d,%d)", b, x, n, sx, sn)
			}
		} else if sn > 0 {
			t.Fatalf("% x: we errored (%v) but stdlib decoded %d", b, err, sx)
		}
	})
}
