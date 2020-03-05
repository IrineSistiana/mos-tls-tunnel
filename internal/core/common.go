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
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
)

var ioCopybuffPool = &sync.Pool{New: func() interface{} {
	return make([]byte, defaultCopyIOBufferSize)
}}

func acquireIOBuf() []byte {
	return ioCopybuffPool.Get().([]byte)
}

func releaseIOBuf(b []byte) {
	ioCopybuffPool.Put(b)
}

type firstErr struct {
	reportrOnce sync.Once
	err         atomic.Value
}

func (fe *firstErr) report(err error) {
	fe.reportrOnce.Do(func() {
		// atomic.Value can't store nil value
		if err != nil {
			fe.err.Store(err)
		}
	})
}

func (fe *firstErr) getErr() error {
	v := fe.err.Load()
	if err, ok := v.(error); ok {
		return err
	}
	return nil
}

// openTunnel opens a tunnel between a and b, if any end
// reports an error during io.Copy, openTunnel will close
// both of them.
func openTunnel(a, b net.Conn, timeout time.Duration) error {
	fe := firstErr{}

	go openOneWayTunnel(a, b, timeout, &fe)
	openOneWayTunnel(b, a, timeout, &fe)

	return fe.getErr()
}

// don not use this func, use openTunnel instead
func openOneWayTunnel(dst, src net.Conn, timeout time.Duration, fe *firstErr) {
	buf := acquireIOBuf()
	_, err := copyBuffer(dst, src, buf, timeout)

	// a nil err is io.EOF err, which is surpressed by copyBuffer.
	// report a nil err means one conn was closed by peer.
	fe.report(err)

	//let another goroutine break from copy loop
	dst.Close()
	src.Close()

	releaseIOBuf(buf)
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

func handleClientMuxConn(smuxConfig *smux.Config, maxStream int, conn net.Conn, handleStream func(net.Conn, *logrus.Entry), requestEntry *logrus.Entry) {
	sess, err := smux.Server(conn, smuxConfig)
	if err != nil {
		requestEntry.Errorf("smux server, %v", err)
	}

	for {
		if sess.IsClosed() {
			return
		}
		stream, err := sess.AcceptStream()
		if err != nil {
			// surpress expected err
			if e, ok := err.(*websocket.CloseError); ok && e.Code == websocket.CloseNormalClosure {
				return
			}
			if err == io.EOF {
				return
			}

			// warn unexpected err
			requestEntry.Warnf("accept smux stream, %v", err)
			return
		}
		if sess.NumStreams() > maxStream {
			stream.Close()
			requestEntry.Warn(ErrTooManyStreams)
			return
		}
		requestEntry.Debug("accepted a smux stream")

		go handleStream(stream, requestEntry)
	}
}
