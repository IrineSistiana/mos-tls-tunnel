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
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/IrineSistiana/mos-tls-tunnel/internal/core"
)

func main() {
	c := new(core.ServerConfig)

	commandLine := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	commandLine.StringVar(&c.BindAddr, "b", "", "[Host:Port] Bind address, e.g. '127.0.0.1:1080'")
	commandLine.StringVar(&c.DstAddr, "d", "", "[Host:Port] Destination address")
	commandLine.StringVar(&c.Cert, "cert", "", "[Path] X509KeyPair cert file")
	commandLine.StringVar(&c.Key, "key", "", "[Path] X509KeyPair key file")

	commandLine.BoolVar(&c.EnableWSS, "wss", false, "Enable WebSocket Secure protocol")
	commandLine.StringVar(&c.WSSPath, "wss-path", "/", "WebSocket path")
	commandLine.StringVar(&c.ServerName, "n", "", "Server name. Use to generate self signed certificate DNSName")
	commandLine.BoolVar(&c.EnableMux, "mux", false, "Enable multiplex")
	//tcp options
	commandLine.DurationVar(&c.Timeout, "timeout", 5*time.Minute, "The idle timeout for connections")
	commandLine.BoolVar(&c.EnableTFO, "fast-open", false, "(Linux kernel 4.11+ only) Enable TCP fast open")

	//debug only
	commandLine.BoolVar(&c.Verbose, "verbose", false, "more log")

	sip003Args, err := core.GetSIP003Args()
	if err != nil {
		logrus.Fatalf("sip003 error: %v", err)
	}

	// overwrite args from env
	if sip003Args != nil {
		c.BindAddr = sip003Args.GetRemoteAddr()
		c.DstAddr = sip003Args.GetLocalAddr()
		c.EnableTFO = sip003Args.TFO

		opts, err := core.FormatSSPluginOptions(sip003Args.SS_PLUGIN_OPTIONS)
		if err != nil {
			logrus.Fatalf("invalid sip003 SS_PLUGIN_OPTIONS: %v", err)
		}
		if err := commandLine.Parse(opts); err != nil {
			logrus.Error(err)
		}
	} else {
		err := commandLine.Parse(os.Args[1:])
		if err != nil {
			logrus.Fatal(err)
		}
	}

	server, err := core.NewServer(c)
	if err != nil {
		logrus.Fatalf("init server failed, %v", err)
	}
	go func() {
		if err := server.Start(); err != nil {
			logrus.Fatalf("server exited, %v", err)
		} else {
			logrus.Printf("server exited")
			os.Exit(0)
		}
	}()

	//wait signals
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
	s := <-osSignals
	logrus.Printf("exiting: signal: %v", s)
	os.Exit(0)
}
