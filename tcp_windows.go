// +build windows

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
	"syscall"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

func getControlFunc(conf *tcpConfig) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		return c.Control(conf.setSockOpt)
	}
}

//TCP_MAXSEG TCP_NODELAY SO_SND/RCVBUF etc..
func (c *tcpConfig) setSockOpt(uintptrFd uintptr) {
	fd := windows.Handle(uintptrFd)
	var err error

	if c.noDelay {
		err = windows.SetsockoptInt(fd, windows.IPPROTO_TCP, windows.TCP_NODELAY, 1)
		if err != nil {
			logrus.Errorf("setsockopt TCP_NODELAY, %v", err)
		}
	}

	if c.sndBuf > 0 {
		err := windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_SNDBUF, c.sndBuf)
		if err != nil {
			logrus.Errorf("setsockopt SO_SNDBUF, %v", err)
		}
	}
	if c.rcvBuf > 0 {
		err := windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_RCVBUF, c.rcvBuf)
		if err != nil {
			logrus.Errorf("setsockopt SO_RCVBUF, %v", err)
		}
	}

	return
}
