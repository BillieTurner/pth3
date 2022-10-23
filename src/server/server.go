package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"pth3/internal/pthelper"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	quic "github.com/lucas-clemente/quic-go"
)

var handlerChan = make(chan int)

var certificatePath = flag.String(
	"certificate",
	"",
	"Path to TLS certificate.",
)
var keyPath = flag.String("key", "", "Path to TLS private key.")
var logFile = flag.String("log-file", "", "Path to log file.")

func handleSession(session quic.Connection, serverInfo *pt.ServerInfo) {
	log.Printf("Opened Quic session with %s", session.RemoteAddr())
	handlerChan <- 1
	defer func() {
		handlerChan <- -1
		// session.CloseWithError(qerr.ApplicationErrorCode(qerr.NoError), "")
		log.Printf("Ended Quic session with %s", session.RemoteAddr())
	}()

	stream, err := session.AcceptStream(context.Background())

	if err != nil {
		log.Printf("Unable to create Quic stream: %s", err)
		return
	}

	log.Printf("Succesfully created Quic stream with %s", session.RemoteAddr())

	log.Printf("Connecting to Onion Router")
	or, err := pt.DialOr(serverInfo, session.RemoteAddr().String(), "quic")

	if err != nil {
		log.Printf("Unable to connect to Onion Router: %s", err)
		return
	}

	defer or.Close()

	pthelper.CopyLoop(stream, or)
}

func acceptLoop(listener quic.Listener, serverInfo *pt.ServerInfo) {
	defer listener.Close()

	for {
		session, err := listener.Accept(context.Background())

		if err != nil {
			log.Printf("Error accepting session: %s", err)
			return
		}

		go handleSession(session, serverInfo)
	}
}

func main() {
	flag.Parse()

	if *logFile != "" {
		file, err := os.OpenFile(
			*logFile,
			os.O_RDWR|os.O_CREATE|os.O_APPEND,
			0600,
		)

		if err != nil {
			log.Fatalf("Unable to open log file: %s", err)
		}

		log.SetOutput(file)
		defer file.Close()
	}
	log.Println("srat pt. 01")

	serverInfo, err := pt.ServerSetup(nil)

	if err != nil {
		log.Fatalf("Unable to setup PT server: %s", err)
	}

	log.Printf("Loading TLS certificate from: %s", *certificatePath)
	log.Printf("Loading TLS private key from: %s", *keyPath)
	certificate, err := tls.LoadX509KeyPair(*certificatePath, *keyPath)

	if err != nil {
		log.Fatalf("Unable to load TLS certificates: %s", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		NextProtos:   []string{"h3"},
	}

	listeners := make([]quic.Listener, 0)

	for _, bindAddr := range serverInfo.Bindaddrs {
		if bindAddr.MethodName == "quic" {
			listener, err := quic.ListenAddr(
				bindAddr.Addr.String(),
				tlsConfig,
				nil,
			)

			if err != nil {
				pt.SmethodError(bindAddr.MethodName, err.Error())
				break
			}

			log.Printf("Started Quic listener: %s", bindAddr.Addr.String())
			go acceptLoop(listener, &serverInfo)

			pt.Smethod(bindAddr.MethodName, listener.Addr())
			listeners = append(listeners, listener)
		} else {
			pt.SmethodError(bindAddr.MethodName, "no such method")
		}
	}
	pt.SmethodsDone()

	pthelper.PtWait(listeners, handlerChan)
}
