package varint_test

import (
"fmt"

"github.com/abhishekkarki/ddia/ch05-encoding-evolution/varint"
)

func ExampleAppendUvarint() {
fmt.Printf("% x\n", varint.AppendUvarint(nil, 300))
fmt.Printf("% x\n", varint.AppendUvarint(nil, 1001))
// Output:
// ac 02
// e9 07
}

func ExampleZigZag() {
for _, x := range []int64{0, -1, 1, -2, 250} {
fmt.Print(varint.ZigZag(x), " ")
}
// Output: 0 1 2 3 500
}
