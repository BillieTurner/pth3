package pthelper

import (
	"encoding/binary"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/lucas-clemente/quic-go"
)

type ListenerCloser interface {
	Close() error
}

const PT_NAME = "pth3"
const ALPN = "h3"

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

func unpack(data []byte) [][]byte {
	rst := make([][]byte, 0)
	for len(data) != 0 {
		dlen := binary.BigEndian.Uint32(data[:4])
		msg := data[8 : dlen+8]
		rst = append(rst, msg)
		data = data[dlen+8:]
	}
	return rst
}

func pack(msg []byte) []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(len(msg)))
	extraData := []byte("0000")
	data := append(bs, extraData...)
	data = append(data, msg...)
	return data
}

func CopyLoop(stream quic.Stream, or net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	buffSize := 1024 * 1024 * 2

	// PT -> OR
	go func() {
		// io.Copy(or, stream)
		buf := make([]byte, buffSize)
		for {
			size, err := stream.Read(buf)
			if err != nil {
				log.Println("stream read error ", err)
				break
			}

			_, err = or.Write(buf[:size])
			if err != nil {
				log.Println("OR write error ", err)
				break
			}

			// msgs := unpack(buf[:size])
			// breakLoop := false
			// for _, msg := range msgs {
			// 	log.Println("rec: ", len(msg))
			// 	_, err = or.Write(msg)
			// 	if err != nil {
			// 		breakLoop = true
			// 		log.Println("err ", err)
			// 		break
			// 	}
			// }
			// if breakLoop {
			// 	break
			// }
		}
		wg.Done()
	}()

	// OR -> PT
	go func() {
		// io.Copy(stream, or)
		buf := make([]byte, buffSize)
		for {
			size, err := or.Read(buf)
			if err != nil {
				log.Println("OR read error ", err)
				break
			}

			// data := pack(buf[:size])
			data := buf[:size]
			_, err = stream.Write(data)
			if err != nil {
				log.Println("stream write error ", err)
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
