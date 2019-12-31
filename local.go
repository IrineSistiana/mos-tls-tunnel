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
	"crypto/tls"
	"log"
	"net"
	"time"

	"golang.org/x/net/websocket"
)

var localWSConfig *websocket.Config
var localTLSConfig *tls.Config

func doLocal() {
	//checking
	if len(*path) == 0 {
		log.Fatal("Fatal: bad url")
	}
	if len(*serverName) == 0 {
		log.Fatal("Fatal: bad server name")
	}

	//init tls config
	localTLSConfig = new(tls.Config)
	localTLSConfig.InsecureSkipVerify = *insecureSkipVerify
	localTLSConfig.ServerName = *serverName
	var err error

	//init ws config
	if *modeWSS {
		localWSConfig, err = websocket.NewConfig("wss://"+*serverName+*path, "https://"+*serverName)
		if err != nil {
			log.Fatalf("Fatal: websocket.NewConfig: %v", err)
		}
		localWSConfig.TlsConfig = localTLSConfig
	}

	listener, err := net.Listen("tcp", *bindAddr)
	if err != nil {
		log.Fatalf("Fatal: net.Listen: %v", err)
	}
	defer listener.Close()
	log.Printf("plugin listen at %s", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Error: listener failed, %v", err)
		}
		go forwardToServer(conn)
	}

}

func forwardToServer(leftConn net.Conn) {
	defer leftConn.Close()

	d := &net.Dialer{Timeout: handShakeTimeout}
	d.Control = getControlFunc()

	rightRawConn, err := d.Dial("tcp", *remoteAddr)
	if err != nil {
		log.Printf("Error: tcp failed to dial, %v", err)
		return
	}
	defer rightRawConn.Close()

	rightRawConn.SetDeadline(time.Now().Add(handShakeTimeout)) //set hand shake timeout
	tlsConn := tls.Client(rightRawConn, localTLSConfig)
	if err := tlsConn.Handshake(); err != nil {
		log.Printf("Error: TLS handshake, %v", err)
		return
	}

	var rightConn net.Conn
	if *modeWSS {
		ws, err := websocket.NewClient(localWSConfig, tlsConn)
		if err != nil {
			log.Printf("Error: ws failed to dial, %v", err)
			return
		}
		defer ws.Close()
		rightConn = ws
	} else {
		rightConn = tlsConn
	}
	rightRawConn.SetDeadline(time.Time{}) //hand shake completed, reset timeout

	go openTunnel(rightConn, leftConn)
	openTunnel(leftConn, rightConn)
}
