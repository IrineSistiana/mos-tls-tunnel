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
	"crypto/tls"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"golang.org/x/sync/singleflight"
)

var localTLSConfig *tls.Config
var wssURL string

func doLocal() {
	//checking
	if len(*path) == 0 {
		logrus.Fatal("bad url")
	}
	if len(*serverName) == 0 { //set serverName as remoteAddr
		serverName = remoteAddr //we already checked len(*remoteAddr) before
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
		logrus.Fatalf("net.Listen: %v", err)
	}
	defer listener.Close()
	logrus.Printf("plugin listen at %s", listener.Addr())

	for {
		leftConn, err := listener.Accept()
		if err != nil {
			logrus.Fatalf("listener failed, %v", err)
		}
		logrus.Debugf("leftConn from %s accepted", leftConn.RemoteAddr())

		go forwardToServer(leftConn)
	}

}

func newRightConn() (net.Conn, error) {
	var rightConn net.Conn
	d := &net.Dialer{
		Control: getControlFunc(),
		Timeout: handShakeTimeout,
	}

	if *modeWSS { // websocket enabled
		conn, err := dialWS(d, wssURL, localTLSConfig)
		if err != nil {
			return nil, err
		}
		rightConn = conn
	} else {
		conn, err := tls.DialWithDialer(d, "tcp", *remoteAddr, localTLSConfig)
		if err != nil {
			return nil, err
		}
		rightConn = conn
	}
	return rightConn, nil
}

type smuxSessPool struct {
	singleflight.Group
	sync.Mutex
	pool sync.Map
}

var defaultSessPool = &smuxSessPool{}

var keySmuxOpenSess = "keySmuxOpenSess"

func (p *smuxSessPool) getAvailableSess() (*smux.Session, error) {
	var availableSess *smux.Session

	try := func(key, value interface{}) bool {
		sess := key.(*smux.Session)
		if sess.IsClosed() {
			p.pool.Delete(sess)
			logrus.Debugf("deleted closed sess %p", sess)
			return true
		}

		if sess.NumStreams() < muxMaxConnPerChannel {
			availableSess = sess
			return false
		}
		return true
	}

	p.pool.Range(try)

	if availableSess == nil {
		sess, err := dialSess()
		if err != nil {
			return nil, err
		}
		p.pool.Store(sess, nil)
		availableSess = sess
	}
	return availableSess, nil
}

func dialSess() (*smux.Session, error) {
	rightConn, err := newRightConn()
	if err != nil {
		return nil, err
	}
	sess, err := smux.Client(rightConn, defaultSmuxConfig)
	if err != nil {
		rightConn.Close()
		return nil, err
	}
	logrus.Debugf("new sess %p opend", sess)
	return sess, nil
}

func (p *smuxSessPool) getStream() (*smux.Stream, error) {
	sess, err := p.getAvailableSess()
	if err != nil {
		return nil, err
	}
	return sess.OpenStream()
}

func forwardToServer(leftConn net.Conn) {
	defer leftConn.Close()
	var rightConn net.Conn
	var err error

	if *mux {
		rightConn, err = defaultSessPool.getStream()
		if err != nil {
			logrus.Errorf("mux getStream: %v", err)
			return
		}
	} else {
		rightConn, err = newRightConn()
		if err != nil {
			logrus.Errorf("connect to remote: %v", err)
			return
		}
	}
	logrus.Debugf("rightConn from %s to %s established", leftConn.RemoteAddr(), rightConn.RemoteAddr())

	go openTunnel(rightConn, leftConn)
	openTunnel(leftConn, rightConn)
}
