package quictool

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConn(t *testing.T) {
	certPath := "../../certs/test.cert"
	keyPath := "../../certs/test.key"
	client, err := GetQuicClient(certPath)
	assert.Nil(t, err)

	addr := "localhost:7777"
	server, err := GetQuicServer(certPath, keyPath, addr)
	assert.Nil(t, err)

	msg := "test"

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for {
			conn, err := server.listener.Accept(context.Background())
			if err != nil {
				fmt.Println(err)
				continue
				// return err
			}
			stream, err := conn.AcceptStream(context.Background())
			if err != nil {
				fmt.Println(err)
				return
			}

			buff := make([]byte, 1024)
			for {
				size, err := stream.Read(buff)
				if err != nil {
					break
				}
				assert.Equal(t, msg, string(buff[:size]))
				wg.Done()
				return
			}
		}
	}()

	err = client.DialAddr(addr)
	assert.Nil(t, err)
	stream, err := client.GetStream()
	assert.Nil(t, err)
	(*stream).Write([]byte(msg))

	wg.Wait()
}
