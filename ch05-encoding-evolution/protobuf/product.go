// Package protobuf encodes and decodes one message, Product, in the
// Protocol Buffers wire format by hand, without generated code.
//
// Schema (product.proto):
//
//	message Product {
//		int id = 1;
//	 string name = 2;
//	 int64 price = 3;
//	}
package protobuf

import "github.com/abhishekkarki/ddia/ch05-encoding-evolution/varint"

// Wire types: the low 3 bits of every key. They tell a reader how to find
// the end of a value, even for a field it does not know.
const (
	WireVarint = 0 // int32, int64, uint32, uint64, bool, enum, sint32/64
	WireI64    = 1 // fixed64, sfixed, double: always 8 bytes
	WireLen    = 2 // string, bytes, embedded messages: varint length + bytes
	WireI32    = 5 // fixed32, sfixed32, float: always 4 bytes
)

// Field number from the .proto schema. These numbers, not the names,
// are what go on the wire, so they must never change.
const (
	fieldID    = 1
	fieldName  = 2
	fieldPrice = 3
)

// Product mirrors the .proto message.
type Product struct {
	ID    int64
	Name  string
	Price int64
}

// AppendKey append the key that starts every field on the wire:
//
// varint(fieldNumber << 3 | wireType)
//
// Shifting left by 3 makes room for the type in the low 3 bits:
// OR puts it there. Fields 1-15 produce a 1-byte key; 16 and up need 2+.
func AppendKey(b []byte, tag int, wireType int) []byte {
	return varint.AppendUvarint(b, uint64(tag)<<3|uint64(wireType))
}

// appendString appends a LEN value: the byte length as a varint, then the
// raw bytes. len() counts byte (UTF-8), not characters, which is what
// protobuf wants.
func appendString(b []byte, s string) []byte {
	b = varint.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}

// EncodeProduct serializes p exactly as proto.Marshal would.
//
// Two proto3 rules make the output byte-identical:
//  1. Zero values (0, "") are not written at all. A reader that finds
//     a field missing uses the default, so writing it would waste bytes.
//  2. Fields are written in field-number order.
//
// int64 is a plain varint of the two's-complement bits (uint64(v)), not
// ZigZag. So -1 costs 10 bytes; that is what sint64 exists for.
func EncodeProduct(p Product) []byte {
	var b []byte

	if p.ID != 0 {
		b = AppendKey(b, fieldID, WireVarint)
		b = varint.AppendUvarint(b, uint64(p.ID))
	}
	if p.Name != "" {
		b = AppendKey(b, fieldName, WireLen)
		b = appendString(b, p.Name)
	}
	if p.Price != 0 {
		b = AppendKey(b, fieldPrice, WireVarint)
		b = varint.AppendUvarint(b, uint64(p.Price))
	}
	return b
}
