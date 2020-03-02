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
	c := new(core.MUServerConfig)

	commandLine := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	commandLine.StringVar(&c.ServerAddr, "b", "", "[Host:Port] Server bind address, e.g. '127.0.0.1:1080'")
	commandLine.StringVar(&c.HTTPControllerAddr, "c", "", "[Host:Port]  Controller address")
	commandLine.BoolVar(&c.EnableMux, "mux", false, "Enable multiplex")
	commandLine.DurationVar(&c.Timeout, "timeout", time.Minute, "The idle timeout for connections")

	cert := commandLine.String("cert", "", "[Path] X509KeyPair cert file")
	key := commandLine.String("key", "", "[Path] X509KeyPair key file")
	forceTLS := commandLine.Bool("force-tls", false, "automatically generates a certificate for listening on HTTPS")

	//debug only
	commandLine.BoolVar(&c.Verbose, "verbose", false, "more log")

	err := commandLine.Parse(os.Args[1:])
	if err != nil {
		logrus.Fatal(err)
	}

	server, err := core.NewMUServer(c)
	if err != nil {
		logrus.Fatalf("init server failed, %v", err)
	}

	//start server
	go func() {
		var do func() error
		switch true {
		case !*forceTLS && len(*key) == 0 && len(*cert) == 0:
			do = server.StartHTTPServer
		case *forceTLS:
			do = func() error { return server.StartTLSServer(*key, *cert) }
		default:
			do = func() error { return server.StartTLSServer(*key, *cert) }
		}

		if err := do(); err != nil {
			logrus.Fatalf("server exited, %v", err)
		} else {
			logrus.Printf("server exited")
			os.Exit(0)
		}
	}()

	//start control
	go func() {
		if err := server.StartController(); err != nil {
			logrus.Fatalf("server control exited, %v", err)
		} else {
			logrus.Printf("server control exited")
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
