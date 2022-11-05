package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"log"
	"net"
	"os"
	"strings"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	quic "github.com/lucas-clemente/quic-go"

	pthelper "pth3/internal/pthelper"
)

var handlerChan = make(chan int)

var logFile = flag.String("log-file", "", "Path to log file.")

var certificatePin = flag.String(
	"certificate-pin",
	"",
	"SHA2-256 pin of the server certificate, encoded in hex.",
)

var publicKeyPin = flag.String(
	"public-key-pin",
	"",
	"SHA2-256 pin of the server public key, encoded in hex.",
)

var hmacKey = flag.String("hmac-key", "", "hmac key")

func matched(b bool) string {
	if b {
		return "matched pinned value"
	} else {
		return "didn't match pinned value"
	}
}

func hash_hex(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func handleClient(connection *pt.SocksConn) {
	handlerChan <- 1
	defer func() {
		handlerChan <- -1
		connection.Close()
		log.Printf("Ending connection to %s", connection.Req.Target)
	}()

	tlsConfig := &tls.Config{
		// Note that we allow an insecure connection to be established, such
		// that we can inspect the certificate from the server and terminate
		// the connection if it doesn't match our pin.
		InsecureSkipVerify: true,
		NextProtos:         []string{pthelper.ALPN},
	}

	log.Printf("Connecting to %s", connection.Req.Target)
	session, err := quic.DialAddr(connection.Req.Target, tlsConfig, nil)

	if err != nil {
		log.Printf("Unable to connect to Quic server: %s", err)
		connection.Reject()
		return
	}

	log.Printf("Connected to %s", connection.Req.Target)
	// defer session.Close(nil)

	// Do SHA2-256 key pin check here.
	pinValid := true
	state := session.ConnectionState()

	for _, peerCertificate := range state.TLS.PeerCertificates {
		// Do public key pinning:
		publicKeyHashHex := hash_hex(peerCertificate.RawSubjectPublicKeyInfo)

		// Do certificate pinning:
		certificateHashHex := hash_hex(peerCertificate.Raw)

		// Do pin check.
		publicKeyPinValid := true

		if *publicKeyPin != "" {
			if strings.ToLower(
				*publicKeyPin,
			) != strings.ToLower(
				publicKeyHashHex,
			) {
				publicKeyPinValid = false
			}
			log.Printf(
				"  Public key:  '%s' %s (SHA2-256)",
				publicKeyHashHex,
				matched(publicKeyPinValid),
			)
		}

		certificatePinValid := true

		if *certificatePin != "" {
			if strings.ToLower(
				*certificatePin,
			) != strings.ToLower(
				certificateHashHex,
			) {
				certificatePinValid = false
			}

			log.Printf(
				"  Certificate: '%s' %s (SHA2-256)",
				certificateHashHex,
				matched(certificatePinValid),
			)
		}

		pinValid = publicKeyPinValid && certificatePinValid

		if !pinValid {
			log.Printf(
				"TLS certificate and/or public key did not match pinned values.",
			)
			return
		}
	}

	stream, err := session.OpenStreamSync(context.Background())

	if err != nil {
		log.Printf("Unable to create Quic stream: %s", err)
		connection.Reject()
		return
	}

	// FIXME: Figure out why Grant() takes an net.TCPAddr, but ignores it?
	err = connection.Grant(nil)

	if err != nil {
		log.Printf("Unable to grant session with %s", session.RemoteAddr())
		return
	}

	log.Printf("Granting session with %s", session.RemoteAddr())
	// copyLoop(stream, connection)
	key, _ := hex.DecodeString(*hmacKey)
	pthelper.CopyLoop(stream, connection, key)
}

func acceptLoop(listener *pt.SocksListener) {
	defer listener.Close()

	for {
		connection, err := listener.AcceptSocks()

		if err != nil {
			// TODO
			// netErr, ok := err.(net.Error)
			// if ok && netErr.Temporary() {
			// 	continue
			// }

			return
		}

		go handleClient(connection)
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

	if *publicKeyPin == "" && *certificatePin == "" {
		log.Fatalf("Certificate and/or public key pin missing.")
	}
	if *hmacKey == "" {
		log.Fatalf("hmac key missing.")
	}

	clientInfo, err := pt.ClientSetup(nil)

	if err != nil {
		log.Fatalf("Unable to setup PT Client")
	}

	if clientInfo.ProxyURL != nil {
		log.Fatalf("Proxy unsupported")
	}

	listeners := make([]net.Listener, 0)

	for _, methodName := range clientInfo.MethodNames {
		if methodName == pthelper.PT_NAME {
			listener, err := pt.ListenSocks("tcp", "127.0.0.1:0")

			if err != nil {
				pt.CmethodError(methodName, err.Error())
				break
			}

			go acceptLoop(listener)

			pt.Cmethod(methodName, listener.Version(), listener.Addr())
			listeners = append(listeners, listener)
		} else {
			pt.CmethodError(methodName, "no such method")
		}
	}
	pt.CmethodsDone()

	pthelper.PtWait(listeners, handlerChan)
}
