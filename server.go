// Copyright (c) 2019-2020 IrineSistiana
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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func doServer() {
	//init tls config
	tlsConfig := new(tls.Config)
	if len(*certFile) == 0 && len(*keyFile) == 0 { //self signed cert
		cers, err := generateCertificate()
		if err != nil {
			logrus.Fatalf("generate certificate: %v", err)
		}
		logrus.Print("WARNING: you are using self-signed certificate")
		tlsConfig.Certificates = cers
	} else {
		cer, err := tls.LoadX509KeyPair(*certFile, *keyFile) //load cert
		if err != nil {
			logrus.Fatalf("failed to load key or cert, %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cer}
	}

	listenConfig := net.ListenConfig{Control: getControlFunc()}
	innerListener, err := listenConfig.Listen(context.Background(), "tcp", *bindAddr)
	if err != nil {
		logrus.Fatalf("tls inner Listener: %v", err)
	}

	listener := tls.NewListener(innerListener, tlsConfig)
	logrus.Printf("plugin listen at %s", listener.Addr())

	if *modeWSS {
		upgrader := websocket.Upgrader{
			HandshakeTimeout: handShakeTimeout,
			ReadBufferSize:   0, // buffers allocated by the HTTP server are used
			WriteBufferSize:  wsBuffSize,
			WriteBufferPool:  wsBuffPool,
		}
		http.Handle(*path, wsHandler{u: upgrader})
		err = http.Serve(listener, nil)
		if err != nil {
			logrus.Fatalf("ListenAndServe: %v", err)
		}
	} else {
		for {
			leftConn, err := listener.Accept()
			if err != nil {
				logrus.Fatalf("listener.Accept: %v", err)
			}
			logrus.Debugf("leftConn from %s accepted", leftConn.RemoteAddr())

			go func(leftConn net.Conn) {
				defer leftConn.Close()
				rightConn, err := net.Dial("tcp", *remoteAddr)
				if err != nil {
					logrus.Errorf("net.Dial, %v", err)
					return
				}
				defer rightConn.Close()
				logrus.Debugf("rightConn from %s to %s established", leftConn.RemoteAddr(), rightConn.RemoteAddr())

				go openTunnel(rightConn, leftConn)
				openTunnel(leftConn, rightConn)
			}(leftConn)
		}
	}
}

type wsHandler struct {
	u websocket.Upgrader
}

// ServeHTTP implements http.Handler interface
func (h wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	leftWSconn, err := h.u.Upgrade(w, r, nil)
	if err != nil {
		logrus.Println(err)
		return
	}
	defer leftWSconn.Close()

	rightConn, err := net.Dial("tcp", *remoteAddr)
	if err != nil {
		logrus.Errorf("tcp failed to dial, %v", err)
		return
	}
	defer rightConn.Close()

	leftconn := &wsConn{conn: leftWSconn}
	go openTunnel(leftconn, rightConn)
	openTunnel(rightConn, leftconn)
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
