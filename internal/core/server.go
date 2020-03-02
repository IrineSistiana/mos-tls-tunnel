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

package core

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	mathRand "math/rand"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
)

//Server represents a Server instance
type Server struct {
	conf *ServerConfig

	tlsConf   *tls.Config
	tcpConfig *tcpConfig

	upgrader websocket.Upgrader

	netDialer *net.Dialer

	listenerLocker sync.Mutex
	listener       net.Listener

	smuxConfig *smux.Config

	log *logrus.Logger
}

func NewServer(c *ServerConfig) (*Server, error) {
	server := new(Server)

	if len(c.BindAddr) == 0 {
		return nil, errors.New("need bind address")
	}

	if len(c.DstAddr) == 0 {
		return nil, errors.New("need destination server address")
	}

	if c.Timeout <= 0 {
		return nil, errors.New("timeout value must at least 1 sec")
	}

	//logger
	server.log = logrus.New()
	if c.Verbose {
		server.log.SetLevel(logrus.DebugLevel)
	} else {
		server.log.SetLevel(logrus.ErrorLevel)
	}

	server.conf = c

	//config
	server.tcpConfig = &tcpConfig{tfo: c.EnableTFO}
	server.tlsConf = new(tls.Config)
	if len(c.Cert) == 0 && len(c.Key) == 0 { //self signed cert
		cers, err := generateCertificate(server.conf.ServerName)
		if err != nil {
			return nil, fmt.Errorf("generate certificate: %v", err)
		}
		logrus.Print("WARNING: you are using a self-signed certificate")
		server.tlsConf.Certificates = cers
	} else {
		cer, err := tls.LoadX509KeyPair(c.Cert, c.Key) //load cert
		if err != nil {
			return nil, fmt.Errorf("failed to load key and cert, %v", err)
		}
		server.tlsConf.Certificates = []tls.Certificate{cer}
	}

	//net dialer
	server.netDialer = &net.Dialer{
		Control: getControlFunc(server.tcpConfig),
		Timeout: defaultHandShakeTimeout,
	}

	//ws upgrader
	server.upgrader = websocket.Upgrader{
		HandshakeTimeout: defaultHandShakeTimeout,
		ReadBufferSize:   0, // buffers allocated by the HTTP server are used
		WriteBufferSize:  0,

		Subprotocols: []string{websocketSubprotocolSmuxON, websocketSubprotocolSmuxOFF},
	}

	server.smuxConfig = defaultSmuxConfig()
	return server, nil
}

func (server *Server) Start() error {
	listenConfig := net.ListenConfig{Control: getControlFunc(server.tcpConfig)}
	listener, err := listenConfig.Listen(context.Background(), "tcp", server.conf.BindAddr)
	if err != nil {
		return fmt.Errorf("tls inner Listener: %v", err)
	}

	server.listenerLocker.Lock()
	server.listener = listener
	server.listenerLocker.Unlock()
	server.log.Printf("plugin listen at %s", listener.Addr())

	if server.conf.EnableWSS {
		httpMux := http.NewServeMux()
		httpMux.Handle(server.conf.WSSPath, server)
		err = http.Serve(tls.NewListener(listener, server.tlsConf), httpMux)
		if err != nil {
			return fmt.Errorf("ListenAndServe: %v", err)
		}
	} else {
		for {
			leftRawConn, err := listener.Accept()
			if err != nil {
				return fmt.Errorf("listener.Accept: %v", err)
			}

			go func() {
				requestEntry := logrus.WithField("client", leftRawConn.RemoteAddr())
				requestEntry.Debug("connection accepted")

				leftConn := tls.Server(leftRawConn, server.tlsConf)
				defer leftConn.Close()
				if err := leftConn.Handshake(); err != nil {
					requestEntry.Errorf("tls handshake: %v", err)
					return
				}

				if server.conf.EnableMux {
					server.handleClientMuxConn(leftConn, requestEntry)
				} else {
					server.handleClientConn(leftConn, requestEntry)
				}
			}()

		}
	}
	return nil
}

//Close shutdown server
func (server *Server) Close() error {
	server.listenerLocker.Lock()
	defer server.listenerLocker.Unlock()
	if server.listener != nil {
		return server.listener.Close()
	}
	return nil
}

func (server *Server) handleClientConn(leftConn net.Conn, requestEntry *logrus.Entry) {
	rightConn, err := server.dialDst()
	if err != nil {
		requestEntry.Errorf("dial dst, %v", err)
	}
	defer rightConn.Close()

	openTunnel(rightConn, leftConn, server.conf.Timeout)
}

func (server *Server) handleClientMuxConn(leftConn net.Conn, requestEntry *logrus.Entry) {
	handleClientMuxConn(server.smuxConfig, defaultSmuxMaxStream, leftConn, server.handleClientConn, requestEntry)
}

// ServeHTTP implements http.Handler interface
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestEntry := logrus.WithField("http_client", r.RemoteAddr)
	requestEntry.Debug("http connection accepted")
	leftWSConn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		requestEntry.Errorf("upgrade http request, %v", err)
		return
	}

	leftConn := wrapWebSocketConn(leftWSConn)

	switch leftWSConn.Subprotocol() {
	case websocketSubprotocolSmuxON:
		server.handleClientMuxConn(leftConn, requestEntry)
	case websocketSubprotocolSmuxOFF:
		server.handleClientConn(leftConn, requestEntry)
	default:
		if server.conf.EnableMux {
			server.handleClientMuxConn(leftConn, requestEntry)
		} else {
			server.handleClientConn(leftConn, requestEntry)
		}
	}
}

func (server *Server) dialDst() (net.Conn, error) {
	return server.netDialer.Dial("tcp", server.conf.DstAddr)
}

func generateCertificate(serverName string) ([]tls.Certificate, error) {
	//priv key
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}

	//serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %v", err)
	}

	// set DNSNames
	if len(serverName) == 0 {
		serverName = randServerName()
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: serverName},
		DNSNames:     []string{serverName},

		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return []tls.Certificate{tlsCert}, nil
}

func randServerName() string {
	return fmt.Sprintf("%s.%s", randStr(mathRand.Intn(5)+3), randStr(mathRand.Intn(3)+1))
}

func randStr(length int) string {
	r := mathRand.New(mathRand.NewSource(time.Now().UnixNano()))
	set := "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = set[r.Intn(len(set))]
	}
	return string(b)
}
