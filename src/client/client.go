package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"strings"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	quic "github.com/lucas-clemente/quic-go"

	pthelper "pth3/internal/pthelper"
)

type ClientArgs struct {
	certPin string
	hmacKey string
}

var handlerChan = make(chan int)
var logFile = flag.String("log-file", "", "Path to log file.")

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

func parserArgs(args pt.Args) (*ClientArgs, error) {
	certPin, ok := args.Get("certificate-pin")
	if !ok || len(certPin) == 0 {
		return nil, errors.New("certificate pin missing")
	}
	hmacKey, ok := args.Get("hmac-key")
	if !ok || len(hmacKey) == 0 {
		return nil, errors.New("hmac key missing")
	}
	return &ClientArgs{
		certPin: certPin,
		hmacKey: hmacKey,
	}, nil
}

func handleClient(connection *pt.SocksConn) {
	handlerChan <- 1
	defer func() {
		handlerChan <- -1
		connection.Close()
		log.Printf("Ending connection to %s", connection.Req.Target)
	}()

	args, err := parserArgs(connection.Req.Args)
	if err != nil {
		log.Printf("Can't parse args: %s", err)
		connection.Reject()
		return
	}

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
		// publicKeyHashHex := hash_hex(peerCertificate.RawSubjectPublicKeyInfo)

		// Do certificate pinning:
		certificateHashHex := hash_hex(peerCertificate.Raw)

		// Do pin check.
		// "SHA2-256 pin of the server public key, encoded in hex."
		publicKeyPinValid := true
		// 	"SHA2-256 pin of the server certificate, encoded in hex.",
		certificatePinValid := true
		if !strings.EqualFold(args.certPin, certificateHashHex) {
			certificatePinValid = false
		}

		log.Printf(
			"  Certificate: '%s' %s (SHA2-256)",
			certificateHashHex,
			matched(certificatePinValid),
		)

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
	key, _ := hex.DecodeString(args.hmacKey)
	pthelper.CopyLoop(stream, connection, key)
}

func acceptLoop(listener *pt.SocksListener) {
	defer listener.Close()

	for {
		conn, err := listener.AcceptSocks()
		if err != nil {
			// TODO
			// netErr, ok := err.(net.Error)
			// if ok && netErr.Temporary() {
			// 	continue
			// }
			return
		}

		go handleClient(conn)
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
