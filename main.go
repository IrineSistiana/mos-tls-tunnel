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
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	bindAddr           = flag.String("b", "127.0.0.1:1080", "Bind address")
	remoteAddr         = flag.String("r", "", "Remote address")
	modeServer         = flag.Bool("s", false, "Server mode")
	modeWSS            = flag.Bool("wss", false, "Enable WebSocket Secure protocol")
	path               = flag.String("path", "/", "WebSocket path")
	keyFile            = flag.String("key", "", "Path to key, used by server mode. If both key and cert is empty, a self signed certificate will be used")
	certFile           = flag.String("cert", "", "Path to cert, used by server mode. If both key and cert is empty, a self signed certificate will be used")
	serverName         = flag.String("n", "", "Server name, used to verify the hostname. It is also included in the client's TLS and WSS handshake to support virtual hosting unless it is an IP address.")
	insecureSkipVerify = flag.Bool("sv", false, "Skip verify, client won't verify the server's certificate chain and host name. In this mode, your connections are susceptible to man-in-the-middle attacks. Use it with caution.")
	buffSize           = flag.Int("buff", 4, "Size of io buffer for each connection (kb)")
	timeout            = flag.Duration("timeout", 5*time.Minute, "the idle timeout for connections (sec)")

	buffPool   *sync.Pool
	wsBuffPool *sync.Pool

	//SIP003 not support flag, dont remove it
	tfo = flag.Bool("fast-open", false, "Not support yet, reserved")
	//SIP003 android
	vpnMode = flag.Bool("V", false, "VPN mode, used in android system only")
)

const (
	handShakeTimeout = time.Second * 10
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

		//parse addtional command from os.Args, such as on android platform
		additional := flag.NewFlagSet("additional", flag.ContinueOnError)
		tfo = additional.Bool("fast-open", false, "")
		vpnMode = additional.Bool("V", false, "")
		additional.Parse(os.Args[1:])

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

	if *timeout <= 0 {
		log.Fatal("Fatal: timeout must at least 1 sec")
	}

	buffPool = &sync.Pool{New: func() interface{} {
		return make([]byte, *buffSize*1024)
	}}

	//this buff pool is used as a websocket write buffPool with size = *buffSize*1024
	wsBuffPool = &sync.Pool{New: func() interface{} {
		return nil
	}}

	log.Printf("plugin starting...")

	if *modeServer {
		go doServer()
	} else {
		go doLocal()
	}

	//wait signals
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	s := <-osSignals
	log.Printf("exiting: signal: %v", s)
	os.Exit(0)
}

func openTunnel(dst, src net.Conn) {
	buf := buffPool.Get().([]byte)
	defer buffPool.Put(buf)

	copyBuffer(dst, src, buf, *timeout)
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
