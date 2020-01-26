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
	"flag"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xtaci/smux"

	"github.com/sirupsen/logrus"
)

var (
	bindAddr           = flag.String("b", "127.0.0.1:1080", "Bind address")
	remoteAddr         = flag.String("r", "", "Remote address")
	modeServer         = flag.Bool("s", false, "Indicate program to run as a server. If absent, run as a client")
	enableWSS          = flag.Bool("wss", false, "Enable WebSocket Secure protocol")
	path               = flag.String("path", "/", "WebSocket path")
	keyFile            = flag.String("key", "", "(Server only) Path to key. If both key and cert is empty, a self signed certificate will be used")
	certFile           = flag.String("cert", "", "(Server only) Path to cert. If both key and cert is empty, a self signed certificate will be used")
	serverName         = flag.String("n", "", "Server name. Client will use it to verify the hostname and to support virtual hosting. Server will use it to generate self signed certificate")
	insecureSkipVerify = flag.Bool("sv", false, "(Client only) Skip verify, client won't verify the server's certificate chain and host name.")
	enableMux          = flag.Bool("mux", false, "Enable multiplex")
	muxMaxStream       = flag.Int("max-stream", 4, "(Client only) The max number of multiplexed streams in one ture TCP connection")
	//tcp options
	timeout          = flag.Duration("timeout", 5*time.Minute, "The idle timeout for connections")
	buffSizeKB       = flag.Int("buff", 0, "The value of maximum socket buffer, tcp_SO_RCVBUF and tcp_SO_SNDBUF, in KB. 0 means using system default")
	enableTCPNoDelay = flag.Bool("no-delay", false, "Enable TCP_NODELAY")
	mss              = flag.Int("mss", 0, "(Linux only) The value of TCP_MAXSEG. 0 means using system default")
	enableTFO        = flag.Bool("fast-open", false, "(Linux kernel 4.11+ only) Enable TCP fast open")

	//SIP003 android, init it later
	vpnMode *bool

	//debug only, used in android system to avoid dns lookup dead loop
	fallbackDNS = flag.String("fallback-dns", "", "use this server instead of system default to resolve host name in -b -r, must be an IP address.")

	//debug only
	verbose = flag.Bool("verbose", false, "more log")
)

//vars that will be initialized later
var (
	//buf things
	buffPool   *sync.Pool
	wsBuffPool *sync.Pool

	wsBuffSize     int
	ioCopyBuffSize int
	tcp_SO_SNDBUF  int
	tcp_SO_RCVBUF  int

	//is program running as a plugin
	modePlugin bool

	//mux config
	defaultSmuxConfig *smux.Config

	//tcp config
	defaultLeftTCPConfig  *tcpConfig
	defaultRightTCPConfig *tcpConfig
)

const (
	handShakeTimeout = time.Second * 10
)

func main() {
	//SS_REMOTE_HOST, SS_REMOTE_PORT, SS_LOCAL_HOST and SS_LOCAL_PORT
	var ssRemote, ssLocal string
	srh, srhOk := os.LookupEnv("SS_REMOTE_HOST")
	srp, srpOk := os.LookupEnv("SS_REMOTE_PORT")
	slh, slhOk := os.LookupEnv("SS_LOCAL_HOST")
	slp, slpOk := os.LookupEnv("SS_LOCAL_PORT")
	spo, spoOk := os.LookupEnv("SS_PLUGIN_OPTIONS")

	//set flags from SS_PLUGIN_OPTIONS
	if spoOk {
		commandLineOption := make([]string, 0)
		op := strings.Split(spo, ";")
		for _, so := range op {
			optionPair := strings.Split(so, "=")
			switch len(optionPair) {
			case 1:
				commandLineOption = append(commandLineOption, "-"+optionPair[0])
			case 2:
				commandLineOption = append(commandLineOption, "-"+optionPair[0], optionPair[1])
			default:
				logrus.Fatalf("invalid option string [%s]", so)
			}
		}
		//from SS_PLUGIN_OPTIONS
		flag.CommandLine.Parse(commandLineOption)
	}

	modePlugin = srhOk || srpOk || slhOk || slpOk || spoOk //if any env exist, we are in plugin mode
	if modePlugin {
		//parse and overwrite addtional commands from os.Args, `fast-open` and `V` (android)
		additional := flag.NewFlagSet("additional", flag.ContinueOnError)
		enableTFO = additional.Bool("fast-open", false, "")
		vpnMode = additional.Bool("V", false, "")
		additional.Parse(os.Args[1:])
	} else {
		//from args
		flag.Parse()
	}

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
		logrus.Fatal("no remote server address")
	}

	if *timeout <= 0 {
		logrus.Fatal("timeout must at least 1 sec")
	}

	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	//set buffers' size
	wsBuffSize = 64 * 1024
	ioCopyBuffSize = 16 * 1024

	// 0 means use system default
	tcp_SO_SNDBUF = *buffSizeKB * 1024
	tcp_SO_RCVBUF = *buffSizeKB * 1024

	defaultSmuxConfig = &smux.Config{
		KeepAliveInterval: 10 * time.Second,
		KeepAliveTimeout:  30 * time.Second,
		MaxFrameSize:      16 * 1024,
		MaxReceiveBuffer:  2048 * 1024,
		MaxStreamBuffer:   512 * 1024,
	}

	localTCPConfig := &tcpConfig{
		tfo:     false,
		noDelay: false,
		mss:     0,
		sndBuf:  64 * 1024,
		rcvBuf:  64 * 1024,
	}

	defaultTCPConfig := &tcpConfig{
		tfo:     *enableTFO,
		noDelay: *enableTCPNoDelay,
		mss:     *mss,
		sndBuf:  tcp_SO_SNDBUF,
		rcvBuf:  tcp_SO_RCVBUF,
	}

	// bind
	addr, err := net.ResolveTCPAddr("tcp", *bindAddr)
	if err != nil {
		logrus.Fatalf("bind addr invalid, %v", err)
	}
	if addr.IP.IsLoopback() {
		defaultLeftTCPConfig = localTCPConfig
	} else {
		defaultLeftTCPConfig = defaultTCPConfig
	}

	// remote
	addr, err = net.ResolveTCPAddr("tcp", *remoteAddr)
	if err != nil {
		logrus.Fatalf("remote addr invalid, %v", err)
	}
	if addr.IP.IsLoopback() {
		defaultRightTCPConfig = localTCPConfig
	} else {
		defaultRightTCPConfig = defaultTCPConfig
	}

	buffPool = &sync.Pool{New: func() interface{} {
		return make([]byte, ioCopyBuffSize)
	}}

	//this buff pool is used as a websocket write buffPool with size = wsBuffSize
	wsBuffPool = &sync.Pool{New: func() interface{} {
		return nil
	}}

	if len(*fallbackDNS) != 0 {
		//set fallback dns server
		if net.ParseIP(*fallbackDNS) == nil { //it's not a IP addr
			logrus.Fatalf("fallback dns server must be an IP addr, got %s", *fallbackDNS)
		}

		//just overwrite net.DefaultResolver
		net.DefaultResolver.PreferGo = true
		net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			d.Control = getControlFunc(defaultRightTCPConfig)
			return d.DialContext(ctx, "tcp", *fallbackDNS)
		}
	}

	logrus.Printf("plugin starting...")

	if *modeServer {
		go doServer()
	} else {
		go doLocal()
	}

	//wait signals
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	s := <-osSignals
	logrus.Printf("exiting: signal: %v", s)
	os.Exit(0)
}

func openTunnel(dst, src net.Conn) {
	buf := buffPool.Get().([]byte)

	copyBuffer(dst, src, buf, *timeout)
	dst.Close()
	src.Close()
	buffPool.Put(buf)
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
