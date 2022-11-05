package pthelper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackUnpack(t *testing.T) {
	msgs := []string{"hello", "world", strings.Repeat("a", 100)}
	key := []byte("1234")
	isVerified := false
	for _, msg := range msgs {
		data := pack([]byte(msg), CHUNK_SIZE, key, isVerified)
		rst, _, err := unpack(data, CHUNK_SIZE, key, isVerified)
		isVerified = true
		assert.Equal(t, msg, string(rst))
		assert.Nil(t, err)
	}
}

func TestUnpackLeftover(t *testing.T) {
	msg := strings.Repeat("a", 1000)
	key := []byte("1234")
	isVerified := false

	pivot := 500
	data := pack([]byte(msg), CHUNK_SIZE, key, isVerified)
	msgs := [][]byte{data[:pivot], data[pivot:]}
	leftOver := make([]byte, 0)
	var rst []byte
	var err error
	for i, encodedData := range msgs {
		data = append(leftOver, encodedData...)
		rst, leftOver, err = unpack(data, CHUNK_SIZE, key, isVerified)
		if i == 0 {
			assert.Equal(t, pivot, len(leftOver))
		} else {
			assert.Equal(t, msg, string(rst))
		}
		assert.Nil(t, err)
	}
}
