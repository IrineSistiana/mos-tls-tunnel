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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
)

var ioCopybuffPool = &sync.Pool{New: func() interface{} {
	return make([]byte, defaultCopyIOBufferSize)
}}

func openTunnel(a, b net.Conn, timeout time.Duration) {
	go openOneWayTunnel(a, b, timeout)
	openOneWayTunnel(b, a, timeout)
}

func openOneWayTunnel(dst, src net.Conn, timeout time.Duration) {
	buf := ioCopybuffPool.Get().([]byte)
	copyBuffer(dst, src, buf, timeout)
	dst.Close()
	src.Close()
	ioCopybuffPool.Put(buf)
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
			requestEntry.Errorf("accept smux stream, %v", err)
			return
		}
		if sess.NumStreams() > maxStream {
			requestEntry.Error(ErrTooManyStreams)
			return
		}
		requestEntry.Debug("accepted a smux stream")

		go handleStream(stream, requestEntry)
	}
}
