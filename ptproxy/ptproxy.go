package main

import (
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var ptName = "pth3"

func main() {
	isClient, err := ptIsClient()
	if err != nil {
		os.Exit(1)
	}

	var listeners []net.Listener
	if isClient {
		listeners = ClientStart()
	} else {
		listeners = ServerStart()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	if os.Getenv("TOR_PT_EXIT_ON_STDIN_CLOSE") == "1" {
		// This environment variable means we should treat EOF on stdin
		// just like SIGTERM: https://bugs.torproject.org/15435.
		go func() {
			io.Copy(io.Discard, os.Stdin)
			sigChan <- syscall.SIGTERM
		}()
	}

	<-sigChan
	for _, ln := range listeners {
		ln.Close()
	}
}
