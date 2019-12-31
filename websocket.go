package main

import (
	"crypto/tls"
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

// internelDial is a wrapper for net.Dial, it set the Dialer.Control and uses *remoteAddr as addr
func internelDial(network, addr string) (net.Conn, error) {
	d := net.Dialer{}
	d.Control = getControlFunc()
	return d.Dial(network, *remoteAddr)
}

func dialWS(urlStr string, tlsConfig *tls.Config) (net.Conn, error) {
	d := websocket.Dialer{
		TLSClientConfig: tlsConfig,
		NetDial:         internelDial,

		ReadBufferSize:   *buffSize * 1024,
		WriteBufferSize:  *buffSize * 1024,
		WriteBufferPool:  wsBuffPool,
		HandshakeTimeout: handShakeTimeout,
	}
	conn, _, err := d.Dial(urlStr, nil)
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
