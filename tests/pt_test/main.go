package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	proxy "golang.org/x/net/proxy"
)

var serverAddr = "127.0.0.1:9999"

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	clientS5PortChan := make(chan string)

	go func() {
		startServer()
		wg.Done()
	}()
	go func() {
		startClient(clientS5PortChan)
		wg.Done()
	}()

	clientS5Port := <-clientS5PortChan
	fmt.Println("clientS5Port ", clientS5Port)
	// handleRequst(&clientS5Port)

	wg.Wait()
}

func handleOutput(bs []byte, portCh chan<- string) {
	// s := strings.TrimSuffix(string(bs), "\n")
	lines := strings.Split(string(bs), "\n")
	for _, line := range lines {
		args := strings.Split(line, " ")
		switch args[0] {
		case "CMETHOD":
			addr := args[3]
			portCh <- addr
			// handleS5Conn(&addr)
			// handleRequst(&addr)
		}
	}
}

func handleRequst(addr *string) {
	proxyUrl, err := url.Parse(fmt.Sprintf("socks://%s", *addr))
	// proxyUrl, err := url.Parse(*addr)
	if err != nil {
		panic(err)
	}

	cl := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		},
		Timeout: 3 * time.Second,
	}

	// resp, err := cl.Get("http://google.com")
	resp, err := cl.Get(fmt.Sprintf("https://%s", serverAddr))
	// if err != nil {
	// 	panic(err)
	// }
	fmt.Println("resp", resp, err)
}

func handleS5Conn(addr *string) {
	// conn, err := net.Dial("tcp", fmt.Sprintf("socks5://%s", *addr))
	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		return
	}
	_, err = conn.Write([]byte("test1"))
	if err != nil {
		log.Println("error1 ", err)
	}
	if err != nil {
		log.Println("error ", err)
	}
}

func startClient(portCh chan<- string) {
	cmds := strings.Split(
		"go run ../../main.go -client -cert=../../certs/test.cert",
		" ",
	)
	cmd := exec.Command(cmds[0], cmds[1:]...)

	envs := []string{
		"TOR_PT_MANAGED_TRANSPORT_VER=1",
		"TOR_PT_CLIENT_TRANSPORTS=pth3",
	}
	cmd.Env = append(os.Environ(), envs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout

	// log.SetOutput()
	buf := make([]byte, 1024)
	go func() {
		for {
			size, err := stdout.Read(buf)
			if err != nil {
				log.Print("error ", err)
				return
				// break
			}
			go handleOutput(buf[:size], portCh)
			fmt.Println(
				"client: ",
				strings.TrimSuffix(string(buf[:size]), "\n"),
			)
		}
	}()
	cmd.Start()
	cmd.Wait()
}

func startServer() {
	cmds := strings.Split(
		"go run ../../main.go -server -cert=../../certs/test.cert -key=../../certs/test.key",
		" ",
	)
	cmd := exec.Command(cmds[0], cmds[1:]...)

	envs := []string{
		"TOR_PT_MANAGED_TRANSPORT_VER=1",
		"TOR_PT_SERVER_TRANSPORTS=pth3",
		fmt.Sprintf("TOR_PT_SERVER_BINDADDR=pth3-%s", serverAddr),
		"TOR_PT_ORPORT=127.0.0.1:9001",
	}
	cmd.Env = append(os.Environ(), envs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout

	// log.SetOutput()
	buf := make([]byte, 1024)
	go func() {
		for {
			size, err := stdout.Read(buf)
			if err != nil {
				log.Print("error ", err)
				return
				// break
			}
			// go handleOutput(buf[:size])
			fmt.Println(
				"server: ",
				strings.TrimSuffix(string(buf[:size]), "\n"),
			)
		}
	}()
	cmd.Start()
	cmd.Wait()
}

func getS5Conn(addr1 *string, addr2 *string) error {
	// addr1 := "127.0.0.1:55555"
	// addr2 := "127.0.0.1:0"
	dialer, err := proxy.SOCKS5(
		"tcp",
		*addr1,
		nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		})
	if err != nil {
		return err
	}
	// conn, err := dialer.Dial("tcp", "127.0.0.1:5555")
	conn, err := dialer.Dial("tcp", *addr2)
	if err != nil {
		// fmt.Println(err)
		return err
	}
	// return &conn, err
	_, err = conn.Write([]byte("foobar"))
	if err != nil {
		return err
	}
	return nil
}
