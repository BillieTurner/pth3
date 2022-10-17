package h3tool

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
)

type H3Client struct {
	roundTripper *http3.RoundTripper
	client       *http.Client
}

func GetH3Client(certPath string) (*H3Client, error) {
	// Read in the cert file
	cert, err := os.ReadFile(certPath)
	if err != nil {
		// log.Fatalf("Failed to append %q to RootCAs: %v", certPath, err)
		return nil, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)

	var qconf quic.Config

	insecure := false
	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: insecure,
			// KeyLogWriter:       keyLog,
		},
		QuicConfig: &qconf,
	}
	// defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
		Timeout:   time.Hour,
	}

	return &H3Client{
		roundTripper: roundTripper,
		client:       hclient,
	}, nil
}
