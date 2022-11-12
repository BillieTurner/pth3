package mincer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMincer(t *testing.T) {
	size := 1000
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i)
	}

	mincer := Mincer{
		MinRate:          0,
		MaxRate:          10,
		ChunkSize:        65,
		MinChunkPerGroup: 4,
		MaxChunkPerGroup: 12,
	}
	require.Nil(t, mincer.Init(nil))
	for c := 0; c < 10; c++ {
		rst := mincer.Run(data)
		newData := make([]byte, 0)
		for i := 0; i < len(rst); i++ {
			newData = append(newData, rst[i]...)
		}
		for i := 0; i < size; i++ {
			require.Equal(t, data[i], newData[i])
		}
	}
}

func BenchmarkMincer(b *testing.B) {
	size := 1000
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i)
	}
	mincer := Mincer{
		MinRate:          0,
		MaxRate:          10,
		ChunkSize:        65,
		MinChunkPerGroup: 4,
		MaxChunkPerGroup: 12,
	}
	mincer.Init(nil)
	for i := 0; i < b.N; i++ {
		mincer.Run(data)
	}
}
