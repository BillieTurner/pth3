package pthelper

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	mincer "pth3/internal/mincer"

	"github.com/lucas-clemente/quic-go"
)

type ListenerCloser interface {
	Close() error
}

const PT_NAME = "pth3"
const ALPN = "h3"
const CHUNK_SIZE = 512

func PtWait[T ListenerCloser](
	listeners []T,
	handlerChan <-chan int,
) {
	numHandlers := 0
	var sig os.Signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	sig = nil
	for sig == nil {
		select {
		case n := <-handlerChan:
			numHandlers += n
		case sig = <-sigChan:
		}
	}

	for _, listener := range listeners {
		listener.Close()
	}

	for n := range handlerChan {
		numHandlers += n
		if numHandlers == 0 {
			break
		}
	}
}

func chunkSlice(slice []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

func getSign(data []byte, key []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(data)
	expectedMAC := mac.Sum(nil)
	return expectedMAC
}

func checkSign(data []byte, sign []byte, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(data)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(sign, expectedMAC)
}

func addHeader(msg []byte, sign []byte, isPadding bool) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(len(msg)))
	extraData := []byte{0b0, 0b0}
	if isPadding {
		extraData = []byte{0b1, 0b0}
	}
	data := append(bs, extraData...)
	if sign != nil {
		data = append(data, sign...)
	}
	data = append(data, msg...)
	return data
}

func genPaddingChunk() []byte {
	bs := make([]byte, 100)
	return bs
}

func isPaddingChunk(bs []byte) bool {
	return bs[0]&0b1 == 1
}

func pack(
	bs []byte,
	size int,
	key []byte,
	isVerified bool,
) []byte {
	chunks := chunkSlice(bs, size)
	var rst []byte
	var sign []byte
	for i, chunk := range chunks {
		if !isVerified && i == 0 {
			sign = getSign(chunk, key)
		} else {
			sign = nil
		}
		c := addHeader(chunk, sign, false)
		rst = append(rst, c...)
	}
	// padding
	// if rand.Intn(100) < 30 {
	// 	padding := genPaddingChunk()
	// 	sign := getSign(padding, key)
	// 	c := addHeader(padding, sign, true)
	// 	rst = append(rst, c...)
	// }
	return rst
}

func unpack(
	data []byte,
	chunkSize int,
	key []byte,
	isVerified bool,
) ([]byte, []byte, error) {
	rst := make([]byte, 0)
	signSize := 20
	headerSize := 6
	headerWithSignSize := signSize + headerSize
	leftOver := make([]byte, 0)
	// [4 size] [2 config] [20 sign (only in first chunk)] [* data]
	for len(data) != 0 {
		if isVerified {
			if len(data) < headerSize {
				leftOver = data
				break
			}
		} else {
			if len(data) < headerWithSignSize {
				leftOver = data
				break
			}
		}

		dlen := int(binary.BigEndian.Uint32(data[:4]))
		if dlen > chunkSize {
			break
		}

		var msg []byte
		var dataSize int
		if isVerified {
			dataSize = dlen + headerSize
			if len(data) < dataSize {
				leftOver = data
				break
			}
			msg = data[headerSize:dataSize]
		} else {
			dataSize = dlen + headerWithSignSize
			sign := data[headerSize:headerWithSignSize]
			if len(data) < dataSize {
				leftOver = data
				break
			}
			msg = data[headerWithSignSize:dataSize]
			if !checkSign(msg, sign, key) {
				return nil, nil, errors.New("invalid signature")
			}
			isVerified = true
		}

		if !isPaddingChunk(data[4:6]) {
			rst = append(rst, msg...)
		}
		data = data[dataSize:]
	}
	return rst, leftOver, nil
}

func CopyLoop(stream quic.Stream, or net.Conn, key []byte) {
	var wg sync.WaitGroup
	wg.Add(2)

	buffSize := 1024 * 1024 * 2

	// PT -> OR
	go func() {
		// io.Copy(or, stream)
		buf := make([]byte, buffSize)
		leftOver := make([]byte, 0)
		isVerified := false
		for {
			size, err := stream.Read(buf)
			if err != nil {
				log.Println("stream read error ", err)
				break
			}

			data := append(leftOver, buf[:size]...)
			data, leftOver, err = unpack(data, CHUNK_SIZE, key, isVerified)
			if err != nil {
				log.Println("invalid signature ", err)
				// stream.Close()
				break
			}
			isVerified = true

			_, err = or.Write(data)
			if err != nil {
				log.Println("OR write error ", err)
				break
			}
		}
		wg.Done()
	}()

	// OR -> PT
	go func() {
		mincerClient := mincer.Mincer{
			MinRate:          0,
			MaxRate:          10,
			ChunkSize:        65,
			MinChunkPerGroup: 4,
			MaxChunkPerGroup: 12,
		}
		mincerClient.Init(nil)
		// io.Copy(stream, or)
		buf := make([]byte, buffSize)
		isVerified := false
		for {
			size, err := or.Read(buf)
			if err != nil {
				log.Println("OR read error ", err)
				break
			}

			data := pack(buf[:size], CHUNK_SIZE, key, isVerified)
			isVerified = true
			exitLoop := false
			chunks := mincerClient.Run(data)
			for i := 0; i < len(chunks); i++ {
				_, err = stream.Write(chunks[i])
				if err != nil {
					log.Println("stream write error ", err)
					exitLoop = true
					break
				}
			}
			if exitLoop {
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
