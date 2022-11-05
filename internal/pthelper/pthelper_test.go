package pthelper

import (
	"fmt"
	"testing"
)

func TestPackUnpack(t *testing.T) {
	msg1 := []byte("hello")
	key := []byte("1234")
	data1 := pack(msg1, CHUNK_SIZE, key)
	fmt.Println("Data ", len(data1), data1)

	rst, _ := unpack(data1, CHUNK_SIZE, key)
	for _, msg := range rst {
		fmt.Println("got ", string(msg))
	}

	msg2 := append(msg1, []byte(" world")...)
	data2 := pack(msg2, CHUNK_SIZE, key)
	fmt.Println("Data ", data2)

	rst, _ = unpack(data2, CHUNK_SIZE, key)
	for _, msg := range rst {
		fmt.Println("got ", string(msg))
	}
}
