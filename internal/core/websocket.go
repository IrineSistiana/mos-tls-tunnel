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

// connection is a wrapper for net.Conn over WebSocket connection.
type wsConn struct {
	conn   *websocket.Conn
	reader io.Reader
}

func dialWebsocketConn(d *websocket.Dialer, url string) (net.Conn, error) {
	conn, _, err := d.Dial(url, nil)
	return &wsConn{conn: conn}, err
}

// Read implements io.Reader.
func (c *wsConn) Read(b []byte) (int, error) {
	var err error
	for {
		//previous reader reach the EOF, get next reader
		if c.reader == nil {
			//always BinaryMessage
			_, c.reader, err = c.conn.NextReader()
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
func (c *wsConn) Write(b []byte) (int, error) {
	if err := c.conn.WriteMessage(websocket.BinaryMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *wsConn) Close() error {
	return c.conn.Close()
}

func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *wsConn) SetDeadline(t time.Time) error {
	return c.conn.UnderlyingConn().SetDeadline(t)
}

func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
