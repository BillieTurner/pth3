package ptproxy

import (
	"log"
	"net"

	"pth3/internal/quictool"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	"github.com/lucas-clemente/quic-go"
)

var ptClientInfo pt.ClientInfo

type PTClient struct {
	listeners *[]PTListener
}

func (p *PTClient) Wait() {
	ptWait(*p.listeners)
}

func cHandler(conn *pt.SocksConn, client *quictool.QuicClient) error {
	defer conn.Close()

	if err := client.DialAddr(conn.Req.Target); err != nil {
		return err
	}

	stream, err := (*client.Conn).OpenStream()
	if err != nil {
		return err
	}

	quit := make(chan bool)
	errChan := make(chan error)

	go S5ToH3(conn.Conn, stream, quit, errChan)
	go H3ToS5(stream, conn.Conn, quit, errChan)

	err = <-errChan
	quit <- true
	return err
}

func cAcceptLoop(ln *pt.SocksListener, client *quictool.QuicClient) error {
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			return err
		}
		go cHandler(conn, client)
		// go cHandler2(conn)
	}
}

func GetClient(certPath string) *PTClient {
	var err error

	client, err := quictool.GetQuicClient(certPath)
	if err != nil {
		log.Fatal(err)
	}

	ptClientInfo, err = pt.ClientSetup(nil)
	if err != nil {
		log.Fatal(err)
	}

	if ptClientInfo.ProxyURL != nil {
		msg := "proxy is not supported"
		pt.ProxyError(msg)
		log.Fatal(msg)
	}

	listeners := make([]PTListener, 0)
	for _, methodName := range ptClientInfo.MethodNames {
		switch methodName {
		case ptName:
			ln, err := pt.ListenSocks("tcp", "127.0.0.1:0")
			if err != nil {
				pt.CmethodError(methodName, err.Error())
				break
			}
			go cAcceptLoop(ln, client)
			pt.Cmethod(methodName, ln.Version(), ln.Addr())
			listeners = append(listeners, ln)
		default:
			pt.CmethodError(methodName, "no such method")
		}
	}
	pt.CmethodsDone()
	return &PTClient{
		listeners: &listeners,
	}
}

func S5ToH3(
	conn net.Conn,
	stream quic.Stream,
	quit <-chan bool,
	errChan chan<- error,
) {
	buff := make([]byte, 1024)
	var size int
	var err error
	for {
		select {
		case <-quit:
			return
		default:
			if size, err = conn.Read(buff); err != nil {
				errChan <- err
				return
			}
			log.Println("got data ", string(buff[:size]))
			if _, err = stream.Write(buff[:size]); err != nil {
				errChan <- err
				return
			}
		}
	}
}

func H3ToS5(
	stream quic.Stream,
	conn net.Conn,
	quit <-chan bool,
	errChan chan<- error,
) {
	buff := make([]byte, 1024)
	var size int
	var err error
	for {
		select {
		case <-quit:
			return
		default:
			if size, err = stream.Read(buff); err != nil {
				errChan <- err
				return
			}
			if _, err = conn.Write(buff[:size]); err != nil {
				errChan <- err
				return
			}
		}
	}
}
