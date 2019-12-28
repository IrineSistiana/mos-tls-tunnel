// Copyright (c) 2019 IrineSistiana
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"

	"golang.org/x/net/websocket"
)

func doServer() {
	//init tls config
	tlsConfig := new(tls.Config)
	if len(*certFile) == 0 && len(*keyFile) == 0 { //self signed cert
		cers, err := generateCertificate()
		if err != nil {
			log.Fatalf("Fatal: generate certificate: %v", err)
		}
		log.Print("WARNING: you are using self-signed certificate")
		tlsConfig.Certificates = cers
	} else {
		cer, err := tls.LoadX509KeyPair(*certFile, *keyFile) //load cert
		if err != nil {
			log.Fatalf("Fatal: failed to load key or cert, %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cer}
	}

	listener, err := tls.Listen("tcp", *bindAddr, tlsConfig)
	if err != nil {
		log.Fatalf("Fatal: tls.Listen: %v", err)
	}
	log.Printf("plugin listen at %s", listener.Addr())

	if *modeWSS {
		http.Handle(*path, websocket.Handler(server))
		err = http.Serve(listener, nil)
		if err != nil {
			log.Fatalf("Fatal: ListenAndServe: %v", err)
		}
	} else {
		for {
			leftConn, err := listener.Accept()
			if err != nil {
				log.Fatalf("Fatal: listener.Accept: %v", err)
			}

			go func(leftConn net.Conn) {
				defer leftConn.Close()
				rightConn, err := net.Dial("tcp", *remoteAddr)
				if err != nil {
					log.Printf("Error: net.Dial, %v", err)
					return
				}
				defer rightConn.Close()

				go openTunnel(rightConn, leftConn)
				openTunnel(leftConn, rightConn)
			}(leftConn)
		}
	}
}

func server(ws *websocket.Conn) {
	defer ws.Close()
	rightConn, err := net.Dial("tcp", *remoteAddr)
	if err != nil {
		log.Printf("Error: tcp failed to dial, %v", err)
		return
	}
	defer rightConn.Close()

	go openTunnel(ws, rightConn)
	openTunnel(rightConn, ws)
}

func generateCertificate() ([]tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	if len(*serverName) != 0 {
		template.DNSNames = []string{*serverName}
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return []tls.Certificate{tlsCert}, nil
}
