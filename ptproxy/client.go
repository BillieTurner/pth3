package main

import (
	"io"
	"net"
	"os"
	"sync"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
)

var ptClientInfo pt.ClientInfo

func clientCopyLoop(a, b net.Conn) {
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

func cHandler(conn *pt.SocksConn) error {
	defer conn.Close()
	remote, err := net.Dial("tcp", conn.Req.Target)
	if err != nil {
		conn.Reject()
		return err
	}
	defer remote.Close()
	err = conn.Grant(remote.RemoteAddr().(*net.TCPAddr))
	if err != nil {
		return err
	}

	clientCopyLoop(conn, remote)

	return nil
}

func cAcceptLoop(ln *pt.SocksListener) error {
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			// e.Temporary()
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			return err
		}
		go cHandler(conn)
	}
}

func ClientStart() []net.Listener {
	var err error

	ptClientInfo, err = pt.ClientSetup(nil)
	if err != nil {
		os.Exit(1)
	}

	if ptClientInfo.ProxyURL != nil {
		pt.ProxyError("proxy is not supported")
		os.Exit(1)
	}

	listeners := make([]net.Listener, 0)
	for _, methodName := range ptClientInfo.MethodNames {
		switch methodName {
		case ptName:
			ln, err := pt.ListenSocks("tcp", "127.0.0.1:0")
			if err != nil {
				pt.CmethodError(methodName, err.Error())
				break
			}
			go cAcceptLoop(ln)
			pt.Cmethod(methodName, ln.Version(), ln.Addr())
			listeners = append(listeners, ln)
		default:
			pt.CmethodError(methodName, "no such method")
		}
	}
	pt.CmethodsDone()
	return listeners
}
