package ptproxy

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var ptName = "pth3"

func ptIsClient() (bool, error) {
	clientEnv := os.Getenv("TOR_PT_CLIENT_TRANSPORTS")
	serverEnv := os.Getenv("TOR_PT_SERVER_TRANSPORTS")
	if clientEnv != "" && serverEnv != "" {
		return false, errors.New(
			"TOR_PT_[CLIENT,SERVER]_TRANSPORTS both set",
		)
	} else if clientEnv != "" {
		return true, nil
	} else if serverEnv != "" {
		return false, nil
	}
	return false, errors.New("not launched as a managed transport")
}

func ptWait(listeners []PTListener) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	if os.Getenv("TOR_PT_EXIT_ON_STDIN_CLOSE") == "1" {
		// This environment variable means we should treat EOF on stdin
		// just like SIGTERM: https://bugs.torproject.org/15435.
		go func() {
			io.Copy(ioutil.Discard, os.Stdin)
			sigChan <- syscall.SIGTERM
		}()
	}

	// wait for a signal
	<-sigChan

	// signal received, shut down
	for _, ln := range listeners {
		ln.Close()
	}
}
