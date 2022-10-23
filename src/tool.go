package ptproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// Setup a bare-bones TLS config for the server
func GenerateTLSConfig(path *string) {
	certPath := fmt.Sprintf("%s/test.cert", *path)
	keyPath := fmt.Sprintf("%s/test.key", *path)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	notBefore := time.Now()
	notAfter := time.Now().Add(time.Hour * 1000)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		// Subject: pkix.Name{
		// 	Organization: []string{"localhost"},
		// },
		NotBefore: notBefore,
		NotAfter:  notAfter,
		DNSNames:  []string{"127.0.0.1:9999"},
		// IPAddresses: []net.IP{addr},
	}
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&key.PublicKey,
		key,
	)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)
	certPEM := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: certDER},
	)

	err = os.WriteFile(keyPath, keyPEM, 0644)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(certPath, certPEM, 0644)
	if err != nil {
		panic(err)
	}
}
