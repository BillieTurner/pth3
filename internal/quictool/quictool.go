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
		ServerName:         "127.0.0.1:9999",
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
