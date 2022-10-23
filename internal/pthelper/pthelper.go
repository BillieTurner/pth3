package pthelper

import (
	"io"
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
		// switch l := listener.(type) {
		// case net.Listener:
		// 	l.Close()
		// case quic.Stream:
		// 	l.Close()
		// }
	}

	for n := range handlerChan {
		numHandlers += n
		if numHandlers == 0 {
			break
		}
	}
}

func CopyLoop(stream quic.Stream, or net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(or, stream)
		wg.Done()
	}()

	go func() {
		io.Copy(stream, or)
		wg.Done()
	}()

	wg.Wait()
}
