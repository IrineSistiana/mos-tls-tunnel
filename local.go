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
)

var localTLSConfig *tls.Config
var wssURL string

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
		wssURL = "wss://" + *serverName + *path
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
	var rightConn net.Conn

	d := &net.Dialer{
		Control: getControlFunc(),
		Timeout: handShakeTimeout,
	}

	if *modeWSS { // websocket enabled
		conn, err := dialWS(d, wssURL, localTLSConfig)
		if err != nil {
			log.Printf("Error: dial wss connection failed, %v", err)
			return
		}
		defer conn.Close()
		rightConn = conn
	} else {
		conn, err := tls.DialWithDialer(d, "tcp", *remoteAddr, localTLSConfig)
		if err != nil {
			log.Printf("Error: failed to establish TLS connection, %v", err)
			return
		}
		defer conn.Close()
		rightConn = conn
	}

	go openTunnel(rightConn, leftConn)
	openTunnel(leftConn, rightConn)
}
