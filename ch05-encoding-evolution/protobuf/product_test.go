package protobuf

import (
	"encoding/hex"
	"strings"
	"testing"
)

// mustHex turns "08 07 12" into []byte{0x08, 0x07, 0x12}.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// Worksheet Part 1, exercises 1–5.
func TestAppendKey(t *testing.T) {
	cases := []struct {
		tag, wt int
		want    string
	}{
		{1, WireVarint, "08"},
		{2, WireLen, "12"},
		{3, WireVarint, "18"},
		{4, WireVarint, "20"},
		{4, WireI64, "21"},
		{4, WireLen, "22"},
		{4, WireI32, "25"},
		{15, WireVarint, "78"},
		{16, WireVarint, "80 01"}, // first field number that needs a 2-byte key
	}
	for _, c := range cases {
		got := AppendKey(nil, c.tag, c.wt)
		if want := mustHex(t, c.want); string(got) != string(want) {
			t.Errorf("AppendKey(%d, %d) = % x, want % x", c.tag, c.wt, got, want)
		}
	}
}

// Worksheet Part 2, exercises 8–11.
func TestEncodeProduct(t *testing.T) {
	cases := []struct {
		name string
		in   Product
		want string
	}{
		{"checklist example", Product{7, "Pen", 250}, "08 07 12 03 50 65 6e 18 fa 01"},
		{"zero ID omitted", Product{0, "Pen", 250}, "12 03 50 65 6e 18 fa 01"},
		{"empty name omitted", Product{300, "", 1}, "08 ac 02 18 01"},
		{"negative int64 costs 10 bytes", Product{1, "A", -1}, "08 01 12 01 41 18 ff ff ff ff ff ff ff ff ff 01"},
		{"all zero is empty", Product{}, ""},
	}
	for _, c := range cases {
		got := EncodeProduct(c.in)
		if want := mustHex(t, c.want); string(got) != string(want) {
			t.Errorf("%s: EncodeProduct(%+v)\n got  % x\n want % x", c.name, c.in, got, want)
		}
	}
}
