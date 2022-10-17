package ptproxy

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"pth3/internal/quictool"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	"github.com/lucas-clemente/quic-go"
)

var ptServerInfo pt.ServerInfo

type PTServer struct {
	listeners *[]PTListener
}

type PTListener interface {
	Close() error
}

func (p *PTServer) Wait() {
	ptWait(*p.listeners)
}

func sHandler(conn quic.Connection) error {
	// defer conn.Close()
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		return err
	}

	or, err := pt.DialOr(&ptServerInfo, conn.RemoteAddr().String(), ptName)
	if err != nil {
		return err
	}
	defer or.Close()

	quit := make(chan bool)
	errChan := make(chan error)
	H3ToS5(stream, or, quit, errChan)
	S5ToH3(or, stream, quit, errChan)

	err = <-errChan
	quit <- true
	return err
}

func sAcceptLoop(ln quic.Listener) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept(context.Background())
		if err != nil {
			// e.Temporary()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			return err
		}
		go sHandler(conn)
	}
}

func GetServer(certPath string, keyPath string, addr string) *PTServer {
	var err error

	ptServerInfo, err = pt.ServerSetup(nil)
	if err != nil {
		os.Exit(1)
	}

	listeners := make([]PTListener, 0)
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		switch bindaddr.MethodName {
		case ptName:
			// ln, err := net.ListenTCP("tcp", bindaddr.Addr)
			// if err != nil {
			// 	pt.SmethodError(bindaddr.MethodName, err.Error())
			// 	break
			// }
			// go sAcceptLoop(ln)
			// pt.Smethod(bindaddr.MethodName, ln.Addr())
			// listeners = append(listeners, ln)
			quicServer, err := quictool.GetQuicServer(certPath, keyPath, addr)
			ln := quicServer.Listener
			if err != nil {
				pt.SmethodError(bindaddr.MethodName, err.Error())
				break
			}
			go sAcceptLoop(ln)
			pt.Smethod(bindaddr.MethodName, ln.Addr())
			listeners = append(listeners, ln)
		default:
			pt.SmethodError(bindaddr.MethodName, "no such method")
		}
	}
	pt.SmethodsDone()
	return &PTServer{
		listeners: &listeners,
	}
}

func ServerStart2() {
	folderPath := flag.String("folder", "", "folder for .cert and .key")
	flag.Parse()

	if _, err := os.Stat(*folderPath); err != nil {
		log.Fatal(err)
	}

	// if len(*folderPath) == 0
	// GenerateTLSConfig(*folderPath)
}
