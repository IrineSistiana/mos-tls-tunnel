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
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
)

type mux struct {
	sync.RWMutex
	pathMap map[string]string

	enableMux bool
	timeout   time.Duration

	upgrader   websocket.Upgrader
	smuxConfig *smux.Config

	log *logrus.Logger
}

func newMux(enableMux bool, timeout time.Duration, logger *logrus.Logger) *mux {
	return &mux{
		pathMap: make(map[string]string),

		enableMux: enableMux,
		timeout:   timeout,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: defaultHandShakeTimeout,
			ReadBufferSize:   0, // buffers allocated by the HTTP server are used
			WriteBufferSize:  0,

			Subprotocols: []string{websocketSubprotocolSmuxON, websocketSubprotocolSmuxOFF},
		},
		smuxConfig: defaultSmuxConfig(),
		log:        logger,
	}
}

func (m *mux) add(a []Args) {
	m.Lock()
	for i := range a {
		m.pathMap[a[i].Path] = a[i].Dst
	}
	m.Unlock()
}

func (m *mux) del(a []Args) {
	m.Lock()
	for i := range a {
		delete(m.pathMap, a[i].Path)
	}
	m.Unlock()
}

func (m *mux) reset() {
	m.Lock()
	m.pathMap = make(map[string]string)
	m.Unlock()
}

func (m *mux) get(path string) (des string, ok bool) {
	m.RLock()
	des, ok = m.pathMap[path]
	m.RUnlock()
	return
}

// ServeHTTP implements http.Handler interface
func (m *mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestEntry := m.log.WithField("client", r.RemoteAddr)
	dst, ok := m.get(r.URL.Path)

	if !ok {
		requestEntry.Warnf("invalid path [%s]", r.URL.Path)
		return
	}

	leftWSConn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		requestEntry.Warnf("upgrade http request failed, %v", err)
		return
	}

	leftConn := wrapWebSocketConn(leftWSConn)
	switch leftWSConn.Subprotocol() {
	case websocketSubprotocolSmuxON:
		m.handleClientMuxConn(leftConn, dst, requestEntry)
	case websocketSubprotocolSmuxOFF:
		m.handleClientConn(leftConn, dst, requestEntry)
	default:
		if m.enableMux {
			m.handleClientMuxConn(leftConn, dst, requestEntry)
		} else {
			m.handleClientConn(leftConn, dst, requestEntry)
		}
	}
}

func (m *mux) handleClientConn(leftConn net.Conn, dst string, requestEntry *logrus.Entry) {
	rightConn, err := net.Dial("tcp", dst)
	if err != nil {
		requestEntry.Warnf("dial dst, %v", err)
		return
	}
	defer rightConn.Close()

	openTunnel(leftConn, rightConn, m.timeout)
}

func (m *mux) handleClientMuxConn(leftConn net.Conn, dst string, requestEntry *logrus.Entry) {
	defer leftConn.Close()

	sess, err := smux.Server(leftConn, m.smuxConfig)
	if err != nil {
		requestEntry.Warnf("smux Server, %v", err)
		return
	}

	for {
		if sess.IsClosed() {
			break
		}
		stream, err := sess.AcceptStream()
		if err != nil {
			requestEntry.Warnf("accept stream, %v", err)
			return
		}
		go m.handleClientConn(stream, dst, requestEntry)
	}
}
