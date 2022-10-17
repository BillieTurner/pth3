package main

import (
	"flag"
	"log"
	"os"

	pth3 "pth3/ptproxy"
)

func main() {
	isClient := flag.Bool("client", false, "client")
	isServer := flag.Bool("server", false, "server")
	isGenCert := flag.Bool("createCert", false, "generate cert")
	certPath := flag.String("cert", "", "cert file path")
	keyPath := flag.String("key", "", "key file path")
	serverAddr := flag.String("serverAddr", "", "server address")
	flag.Parse()

	// logFile, err := os.OpenFile(
	// 	"testlogfile",
	// 	os.O_RDWR|os.O_CREATE|os.O_APPEND,
	// 	0666,
	// )
	// if err != nil {
	// 	log.Fatalf("error opening file: %v", err)
	// }
	// defer logFile.Close()
	// log.SetOutput(logFile)

	// log.Println("cmd ", *isClient, *isServer, *certPath, *isGenCert, *keyPath)
	if *isGenCert {
		// generate certs
		return
	}

	if _, err := os.Stat(*certPath); err != nil {
		log.Fatal("can't find cert file", err)
	}
	if *isClient {
		client := pth3.GetClient(*certPath)
		client.Wait()
	} else if *isServer {
		if _, err := os.Stat(*keyPath); err != nil {
			log.Fatal("can't find key file", err)
		}
		server := pth3.GetServer(*certPath, *keyPath, *serverAddr)
		server.Wait()
	}
}
