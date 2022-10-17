package quictool

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"

	"github.com/lucas-clemente/quic-go"
)

const alpn = "h3"

type QuicClient struct {
	tlsConf tls.Config
	Conn    *quic.Connection
}

type QuicServer struct {
	Listener quic.Listener
}

func GetQuicClient(certPath string) (*QuicClient, error) {
	// Read in the cert file
	cert, err := os.ReadFile(certPath)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(cert); !ok {
		log.Fatalf("Failed to append %q to RootCAs: %v", certPath, err)
		return nil, err
	}

	tlsConf := tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: false,
		NextProtos:         []string{alpn},
	}

	return &QuicClient{
		tlsConf: tlsConf,
	}, nil
}

func (q *QuicClient) DialAddr(addr string) error {
	var qconf quic.Config
	conn, err := quic.DialAddr(addr, &q.tlsConf, &qconf)
	if err != nil {
		return err
	}
	q.Conn = &conn

	return nil
}

func (q *QuicClient) GetStream() (*quic.Stream, error) {
	stream, err := (*q.Conn).OpenStream()
	if err != nil {
		return nil, err
	}
	return &stream, nil
}

func GetQuicServer(
	certPath string,
	keyPath string,
	addr string,
) (*QuicServer, error) {
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	privateKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(cert, privateKey)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{alpn},
	}

	listener, err := quic.ListenAddr(addr, tlsConfig, nil)
	if err != nil {
		return nil, err
	}

	return &QuicServer{
		Listener: listener,
	}, nil
	// return listener, nil
}

// func Temp() error {
// 	localCertFile := "certs/test.cert"
// 	// Read in the cert file
// 	cert, err := os.ReadFile(localCertFile)
// 	if err != nil {
// 		log.Fatalf("Failed to append %q to RootCAs: %v", localCertFile, err)
// 	}
// 	fmt.Println("cert ", cert)
// 	pool := x509.NewCertPool()
// 	pool.AppendCertsFromPEM(cert)

// 	tlsConf := &tls.Config{
// 		RootCAs:            pool,
// 		InsecureSkipVerify: false,
// 		NextProtos:         []string{"h3"},
// 		// NextProtos:         []string{"quic-echo-example"},
// 	}
// 	fmt.Println("tls", tlsConf)
// 	addr := "localhost:4242"
// 	conn, err := quic.DialAddr(addr, tlsConf, nil)
// 	if err != nil {
// 		return err
// 	}

// 	stream, err := conn.OpenStreamSync(context.Background())
// 	if err != nil {
// 		return err
// 	}

// 	message := "foobar"
// 	fmt.Printf("Client: Sending '%s'\n", message)
// 	_, err = stream.Write([]byte(message))
// 	if err != nil {
// 		return err
// 	}

// 	buf := make([]byte, len(message))
// 	_, err = io.ReadFull(stream, buf)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Printf("Client: Got '%s'\n", buf)
// 	return nil
// }
