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
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
)

const (
	muxCheckIdleInterval = time.Second * 2
	muxSessIdleTimeout   = time.Second * 10
)

//Client represents a client instance
type Client struct {
	conf *ClientConfig

	tlsConf   *tls.Config
	tcpConfig *tcpConfig

	wssURL   string
	wsDialer *websocket.Dialer
	dialWSS  func() (net.Conn, error)
	dialTLS  func() (net.Conn, error)

	netDialer *net.Dialer

	ioCopybuffPool *sync.Pool
	wsBuffPool     *sync.Pool

	smuxSessPool smuxSessPool

	listenerLocker sync.Mutex
	listener       net.Listener

	log *logrus.Logger
}

// NewClient inits a client instance
func NewClient(c *ClientConfig) (*Client, error) {
	client := new(Client)

	if len(c.BindAddr) == 0 {
		return nil, errors.New("need bind address")
	}

	if len(c.RemoteAddr) == 0 {
		return nil, errors.New("need remote server address")
	}

	if c.Timeout <= 0 {
		return nil, errors.New("timeout value must at least 1 sec")
	}

	if len(c.ServerName) == 0 { //set serverName as remoteAddr
		c.ServerName = c.RemoteAddr
	}

	//init

	//logger
	client.log = logrus.New()
	if c.Verbose {
		client.log.SetLevel(logrus.DebugLevel)
	} else {
		client.log.SetLevel(logrus.ErrorLevel)
	}

	//buffer pool
	client.ioCopybuffPool = &sync.Pool{New: func() interface{} {
		return make([]byte, defaultCopyIOBufferSize)
	}}

	client.wsBuffPool = &sync.Pool{New: func() interface{} {
		return nil
	}}

	//config
	client.tcpConfig = &tcpConfig{tfo: c.EnableTFO, vpnMode: c.VpnMode}
	client.tlsConf = &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify, ServerName: c.ServerName}

	//net dialer
	client.netDialer = &net.Dialer{
		Control: getControlFunc(client.tcpConfig),
		Timeout: defaultHandShakeTimeout,
	}

	//ws
	client.wssURL = "wss://" + c.ServerName + c.WSSPath
	internelDial := func(network, addr string) (net.Conn, error) {
		// overwrite url host addr
		return client.netDialer.Dial(network, c.RemoteAddr)
	}
	client.wsDialer = &websocket.Dialer{
		TLSClientConfig: client.tlsConf,
		NetDial:         internelDial,

		ReadBufferSize:   defaultWSIOBufferSize,
		WriteBufferSize:  defaultWSIOBufferSize,
		WriteBufferPool:  client.wsBuffPool,
		HandshakeTimeout: defaultHandShakeTimeout,
	}
	client.dialWSS = func() (net.Conn, error) { return dialWebsocketConn(client.wsDialer, client.wssURL) }

	//tls dial
	client.dialTLS = func() (net.Conn, error) {
		return tls.DialWithDialer(client.netDialer, "tcp", c.RemoteAddr, client.tlsConf)
	}

	// fallback dns
	if len(c.FallbackDNS) != 0 {
		//set fallback dns server
		if net.ParseIP(c.FallbackDNS) == nil { //it's not a IP addr
			return nil, fmt.Errorf("fallback dns server must be an IP addr, got %s", c.FallbackDNS)
		}

		//just overwrite net.DefaultResolver
		net.DefaultResolver.PreferGo = true
		net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			d.Control = getControlFunc(client.tcpConfig)
			return d.DialContext(ctx, "tcp", c.FallbackDNS)
		}
	}

	//smux pool
	client.smuxSessPool = smuxSessPool{}

	client.conf = c
	return client, nil
}

var localTLSConfig *tls.Config
var wssURL string

//Start starts the client, it block
func (client *Client) Start() error {
	listenConfig := net.ListenConfig{Control: getControlFunc(client.tcpConfig)}
	listener, err := listenConfig.Listen(context.Background(), "tcp", client.conf.BindAddr)
	if err != nil {
		return fmt.Errorf("net.Listen: %v", err)
	}
	defer listener.Close()
	client.listenerLocker.Lock()
	client.listener = listener
	client.listenerLocker.Unlock()
	client.log.Printf("plugin listen at %s", listener.Addr())

	for {
		leftConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("listener.Accept: %v", err)
		}
		client.log.Debugf("leftConn from %s accepted", leftConn.RemoteAddr())

		go client.forwardToServer(leftConn)
	}
}

//Close shutdown client
func (client *Client) Close() error {
	client.listenerLocker.Lock()
	defer client.listenerLocker.Unlock()
	if client.listener != nil {
		return client.listener.Close()
	}
	return nil
}

func (client *Client) newServerConn() (net.Conn, error) {
	if client.conf.EnableWSS {
		return client.dialWSS()
	}
	return client.dialTLS()
}

type smuxSessPool struct {
	sync.Map
}

func (client *Client) dialNewSmuxSess() (*smux.Session, error) {
	rightConn, err := client.newServerConn()
	if err != nil {
		return nil, err
	}
	sess, err := smux.Client(rightConn, defaultSmuxConfig)
	if err != nil {
		rightConn.Close()
		return nil, err
	}

	// this go routine closes sess if it has been idle for a long time
	go func() {
		ticker := time.NewTicker(muxCheckIdleInterval)
		defer ticker.Stop()
		lastBusy := time.Now()
		for {
			if sess.IsClosed() {
				return
			}

			select {
			case now := <-ticker.C:
				if sess.NumStreams() > 0 {
					lastBusy = now
					continue
				}

				if now.Sub(lastBusy) > muxSessIdleTimeout {
					sess.Close()
					client.smuxSessPool.Delete(sess)
					client.log.Debugf("sess %p closed and deleted, idle timeout", sess)
					return
				}
			}
		}
	}()

	client.smuxSessPool.Store(sess, nil)
	client.log.Debugf("new sess %p opend", sess)
	return sess, nil
}

func (client *Client) getSmuxStream() (*smux.Stream, error) {
	var stream *smux.Stream

	try := func(key, value interface{}) bool {
		sess := key.(*smux.Session)
		if sess.IsClosed() {
			client.smuxSessPool.Delete(sess)
			client.log.Debugf("deleted closed sess %p", sess)
			return true
		}

		if sess.NumStreams() < client.conf.MuxMaxStream {
			// try
			var er error
			stream, er = sess.OpenStream()
			if er != nil {
				client.smuxSessPool.Delete(sess)
				client.log.Errorf("deleted err sess %p: open stream: %v", sess, er)
				return true
			}
			return false
		}
		return true
	}

	client.smuxSessPool.Range(try)

	if stream == nil {
		sess, err := client.dialNewSmuxSess()
		if err != nil {
			return nil, err
		}
		return sess.OpenStream()
	}
	return stream, nil
}

func (client *Client) forwardToServer(leftConn net.Conn) {
	defer leftConn.Close()
	var rightConn net.Conn
	var err error

	if client.conf.EnableMux {
		rightConn, err = client.getSmuxStream()
		if err != nil {
			client.log.Errorf("mux getStream: %v", err)
			return
		}
	} else {
		rightConn, err = client.newServerConn()
		if err != nil {
			client.log.Errorf("connect to remote: %v", err)
			return
		}
	}
	client.log.Debugf("rightConn from %s to %s established", leftConn.RemoteAddr(), rightConn.RemoteAddr())

	go openTunnel(rightConn, leftConn, client.ioCopybuffPool, client.conf.Timeout)
	openTunnel(leftConn, rightConn, client.ioCopybuffPool, client.conf.Timeout)
}
