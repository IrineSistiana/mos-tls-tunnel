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
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketFormatCloseMessage = websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
)

// webSocketConnWrapper is a wrapper for net.Conn over WebSocket connection.
type webSocketConnWrapper struct {
	ws     *websocket.Conn
	reader io.Reader
}

func wrapWebSocketConn(c *websocket.Conn) net.Conn {
	return &webSocketConnWrapper{ws: c}
}

func dialWebsocketConn(d *websocket.Dialer, url string) (net.Conn, error) {
	c, _, err := d.Dial(url, nil)
	return wrapWebSocketConn(c), err
}

// Read implements io.Reader.
func (c *webSocketConnWrapper) Read(b []byte) (int, error) {
	var err error
	for {
		//previous reader reach the EOF, get next reader
		if c.reader == nil {
			//always BinaryMessage
			_, c.reader, err = c.ws.NextReader()
			if err != nil {
				return 0, err
			}
		}

		n, err := c.reader.Read(b)
		if n == 0 && err == io.EOF {
			c.reader = nil
			continue //nothing left in this reader
		}
		return n, err
	}
}

// Write implements io.Writer.
func (c *webSocketConnWrapper) Write(b []byte) (int, error) {
	if err := c.ws.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *webSocketConnWrapper) Close() error {
	c.ws.WriteMessage(websocket.CloseMessage, websocketFormatCloseMessage)
	return c.ws.Close()
}

func (c *webSocketConnWrapper) LocalAddr() net.Addr {
	return c.ws.LocalAddr()
}

func (c *webSocketConnWrapper) RemoteAddr() net.Addr {
	return c.ws.RemoteAddr()
}

func (c *webSocketConnWrapper) SetDeadline(t time.Time) error {
	return c.ws.UnderlyingConn().SetDeadline(t)
}

func (c *webSocketConnWrapper) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *webSocketConnWrapper) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}
