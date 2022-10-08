package main

import (
	"io"
	"net"
	"os"
	"sync"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
)

var ptServerInfo pt.ServerInfo

func serverCopyLoop(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(b, a)
		wg.Done()
	}()
	go func() {
		io.Copy(a, b)
		wg.Done()
	}()

	wg.Wait()
}

func sHandler(conn net.Conn) error {
	defer conn.Close()

	or, err := pt.DialOr(&ptServerInfo, conn.RemoteAddr().String(), ptName)
	if err != nil {
		return err
	}
	defer or.Close()

	serverCopyLoop(conn, or)

	return nil
}

func sAcceptLoop(ln net.Listener) error {
	defer ln.Close()
	for {
		conn, err := ln.Accept()
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

func ServerStart() []net.Listener {
	var err error

	ptServerInfo, err = pt.ServerSetup(nil)
	if err != nil {
		os.Exit(1)
	}

	listeners := make([]net.Listener, 0)
	for _, bindaddr := range ptServerInfo.Bindaddrs {
		switch bindaddr.MethodName {
		case ptName:
			ln, err := net.ListenTCP("tcp", bindaddr.Addr)
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
	return listeners
}
