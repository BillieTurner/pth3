package pthelper

import (
	"fmt"
	"testing"
)

func TestPackUnpack(t *testing.T) {
	msg1 := []byte("hello")
	data1 := pack(msg1)
	fmt.Println("Data ", data1)

	rst := unpack(data1)
	for _, msg := range rst {
		fmt.Println("got ", string(msg))
	}

	msg2 := append(msg1, []byte("world")...)
	data2 := pack(msg2)
	fmt.Println("Data ", data2)

	rst = unpack(data2)
	for _, msg := range rst {
		fmt.Println("got ", string(msg))
	}
}
