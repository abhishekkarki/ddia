// Package varint implements the two primitives that Protocol Buffers,
// Thrift CompactProtocol and Avro all build on (DDIA, "Formats for
// Encoding Data"): base-128 variable-length integers and ZigZag encoding.
package varint

import "errors"

// MaxLen64 is the most bytes a uint64 can need: ceil(64 / 7) = 10.
const MaxLen64 = 10

var (
	// ErrTurncated: the input enable while a byte still had its
	// continuation bit (0x80) set, so the number is incomplete.
	ErrTruncated = errors.New("varint: turncated input")
	// ErrOverflow: the encoded numner does not fit in 64 bits.
	ErrOverflow = errors.New("varint: value overflows uint64")
)

// AppendUvarint appends x to buf as a base-128 varint and returns the
// extended slice.
//
// Each output byte carries 7 bits of payload, least-significant group
// first. The top bit (0x80) is a continuation flag: 1 means "another byte
// follows", 0 means "this is the last byte".
//
// 300 = 0b1_0010_1100
//
//		low 7 bits 0101100 -> with continuation bit: 1_0101100 = 0xac
//	  next 7 bits 00000010 -> last byte, no flag: 0_0000010 = 0x02
//
// => ac 02
func AppendUvarint(buf []byte, x uint64) []byte {
	for x >= 0x80 { // more than 7 bits left -> not the last byte
		buf = append(buf, byte(x)|0x80) // low 7 bits + continuation flag
		x >>= 7
	}
	return append(buf, byte(x)) // final byte: top bit is 0
}

// ReadUvarint decodes a varint from the start of b. It returns the value
// and the number of bytes consumed. It never panics: malformed input
// produces an error instead.
func ReadUvarint(b []byte) (x uint64, n int, err error) {
	var shift uint
	for i, c := range b {
		if i == MaxLen64 {
			// 10 bytes already consumed and still continuing.
			return 0, 0, ErrOverflow
		}
		if c < 0x80 { // last byte
			// The 10th byte sits at shift 63, so only its lowest bit
			// fits into a uint64. Anything bigger is overflow.
			if i == MaxLen64-1 && c > 1 {
				return 0, 0, ErrOverflow
			}
			return x | uint64(c)<<shift, i + 1, nil
		}
		x |= uint64(c&0x7f) << shift
		shift += 7
	}
	// Rin out of bytes while the continuation bit was still set
	// (or b was empty).
	return 0, 0, ErrTruncated
}

// ZigZag maps signed integers to unsigned ones so that values close to
// zero (positive or negative) get small encoding:
//
// 0 -> 0, -1 -> 1, 1 -> 2, -2 -> 3, 2 -> 4, ...
//
// Without it, -1 as a uint64 is 0xffff_ffff_ffff_ffff and costs 10bytes.
//
// x<<1 makes roome for the sign in bit 0. x>>63 is an arithmetic shift:
// all 0s for x >=0, all 1s for x < 0, so XOR-ing with it flips every bit
// of negative numbers (turning -1 = ...1110 after the shift into ...0001).
func ZigZag(x int64) uint64 {
	return uint64(x<<1) ^ uint64(x>>63)
}

// UnZigZag inverts ZigZag. u>>1 recovers the magnitude bits; -(u&1) is
// all 1s if the sign bit was set, so XOR flips them back.
func UnZigZag(u uint64) int64 {
	return int64(u>>1) ^ -int64(u&1)
}

// AppendVarint encodes a signed integet: ZigZag, then varint
// This is exactly what protobuf's sint64 and Avro's long do.
func AppendVarint(buf []byte, x int64) []byte {
	return AppendUvarint(buf, ZigZag(x))
}

// ReadVarint decode a value written by AppendVarint
func ReadVarint(b []byte) (int64, int, error) {
	u, n, err := ReadUvarint(b)
	return UnZigZag(u), n, err
}
