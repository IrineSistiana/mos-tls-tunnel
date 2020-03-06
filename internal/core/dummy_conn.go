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
)

type dummyDialerListener struct {
	notifyListener chan net.Conn
	closeOnce      sync.Once
	isClosed       chan struct{}
}

func newDummyDialerListener() *dummyDialerListener {
	return &dummyDialerListener{
		notifyListener: make(chan net.Conn, 0),
		isClosed:       make(chan struct{}, 0),
	}
}

func (d *dummyDialerListener) Dial(network string, address string) (net.Conn, error) {
	return d.connect()
}

func (d *dummyDialerListener) connect() (net.Conn, error) {
	c1, c2 := net.Pipe()
	select {
	case d.notifyListener <- c2:
		return c1, nil
	case <-d.isClosed:
		return nil, io.ErrClosedPipe
	}
}

func (d *dummyDialerListener) Accept() (net.Conn, error) {
	select {
	case c := <-d.notifyListener:
		return c, nil
	case <-d.isClosed:
		return nil, io.ErrClosedPipe
	}
}

func (d *dummyDialerListener) Addr() net.Addr {
	return pipeAddr{}
}

func (d *dummyDialerListener) Close() error {
	d.closeOnce.Do(func() { close(d.isClosed) })
	return nil
}

// steal from golang net pipeAddr
type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "pipe" }
