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
	"flag"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	bindAddr           = flag.String("b", "127.0.0.1:1080", "bind address")
	remoteAddr         = flag.String("r", "", "remote address")
	modeServer         = flag.Bool("s", false, "server mode")
	keyFile            = flag.String("key", "", "path to key, used by server mode, if both key and cert is empty, a self signed certificate will be used")
	certFile           = flag.String("cert", "", "path to cert, used by server mode, if both key and cert is empty, a self signed certificate will be used")
	serverName         = flag.String("n", "", "server name, used to verify the hostname. it is also included in the client's handshake to support virtual hosting unless it is an IP address.")
	insecureSkipVerify = flag.Bool("sv", false, "skip verifies the server's certificate chain and host name. use it with caution.")
	buffSize           = flag.Int("buff", 32, "size of io buffer for each connection (kb)")
	timeoutInt         = flag.Int("timeout", 300, "read and write timeout deadline(sec)")
	verbose            = flag.Bool("verbose", false, "moooore log")

	tlsConfig tls.Config
	buffPool  *sync.Pool
	timeout   time.Duration
)

func main() {
	//SS_PLUGIN_OPTIONS
	if o, ok := os.LookupEnv("SS_PLUGIN_OPTIONS"); ok {
		commandLineOption := make([]string, 0)
		op := strings.Split(o, ";")
		for _, so := range op {
			optionPair := strings.Split(so, "=")
			switch len(optionPair) {
			case 1:
				commandLineOption = append(commandLineOption, "-"+optionPair[0])
			case 2:
				commandLineOption = append(commandLineOption, "-"+optionPair[0], optionPair[1])
			default:
				log.Fatalf("invalid option string [%s]", so)
			}
		}
		//from SS_PLUGIN_OPTIONS
		flag.CommandLine.Parse(commandLineOption)
	} else {
		//from args
		flag.Parse()
	}

	//SS_REMOTE_HOST, SS_REMOTE_PORT, SS_LOCAL_HOST and SS_LOCAL_PORT
	var ssRemote, ssLocal string
	srh, srhOk := os.LookupEnv("SS_REMOTE_HOST")
	srp, srpOk := os.LookupEnv("SS_REMOTE_PORT")
	slh, slhOk := os.LookupEnv("SS_LOCAL_HOST")
	slp, slpOk := os.LookupEnv("SS_LOCAL_PORT")
	if srhOk && srpOk && slhOk && slpOk {
		ssRemote = net.JoinHostPort(srh, srp)
		ssLocal = net.JoinHostPort(slh, slp)
		if *modeServer { //overwrite flags from env
			remoteAddr = &ssLocal
			bindAddr = &ssRemote
		} else {
			remoteAddr = &ssRemote
			bindAddr = &ssLocal
		}
	}

	if len(*remoteAddr) == 0 {
		log.Fatal("Fatal: no remote server address")
	}

	if *buffSize <= 0 {
		log.Fatal("Fatal: size of io buffer must at least 1kb")
	}

	if *timeoutInt <= 0 {
		log.Fatal("Fatal: timeout must at least 1 sec")
	}

	timeout = time.Duration(*timeoutInt) * time.Second
	buffPool = &sync.Pool{New: func() interface{} {
		return make([]byte, *buffSize*1024)
	}}

	var listener net.Listener
	tlsConfig = tls.Config{
		ServerName:         *serverName,
		InsecureSkipVerify: *insecureSkipVerify,
	}
	if *modeServer {
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

		var err error
		listener, err = tls.Listen("tcp", *bindAddr, &tlsConfig)
		if err != nil {
			log.Fatalf("Fatal:tls.Listen: %v", err)
		}
	} else {
		var err error
		listener, err = net.Listen("tcp", *bindAddr)
		if err != nil {
			log.Fatalf("Fatal: net.Listen: %v", err)
		}
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error: listener failed, %v", err)
				return
			}

			if *verbose {
				log.Printf("accepted an incoming connection from %v", conn.RemoteAddr())
			}

			go forwardLoop(conn)
		}
	}()

	//wait signals
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	s := <-osSignals
	log.Printf("exiting: signal: %v", s)
	os.Exit(0)
}

func forwardLoop(leftConn net.Conn) {
	defer leftConn.Close()

	var rightConn net.Conn
	var err error
	if *modeServer {
		rightConn, err = net.Dial("tcp", *remoteAddr)
	} else {
		rightConn, err = tls.Dial("tcp", *remoteAddr, &tlsConfig)
	}
	if err != nil {
		log.Printf("Error: tls failed to dial, %v", err)
		return
	}
	defer rightConn.Close()

	if *verbose {
		log.Printf("opened a tls tunnel: %v -> %v", leftConn.RemoteAddr(), rightConn.LocalAddr())
	}

	go openTunnel(rightConn, leftConn)
	openTunnel(leftConn, rightConn)
}

func openTunnel(dst, src net.Conn) {
	buf := buffPool.Get().([]byte)
	defer buffPool.Put(buf)

	i, err := copyBuffer(dst, src, buf, timeout)
	if *verbose {
		if err != nil {
			log.Printf("tunnel %v -> %v closed, data exchanged %d, err %v", src.RemoteAddr(), dst.RemoteAddr(), i, err)
		} else {
			log.Printf("tunnel %v -> %v closed, data exchanged %d", src.RemoteAddr(), dst.RemoteAddr(), i)
		}
	}
	dst.Close()
	src.Close()

}

func copyBuffer(dst net.Conn, src net.Conn, buf []byte, timeout time.Duration) (written int64, err error) {

	if len(buf) <= 0 {
		panic("buf size <= 0")
	}

	for {
		src.SetReadDeadline(time.Now().Add(timeout))
		nr, er := src.Read(buf)
		if nr > 0 {
			dst.SetWriteDeadline(time.Now().Add(timeout))
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func generateCertificate() ([]tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
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
