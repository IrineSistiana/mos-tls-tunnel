// +build linux android

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
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

//TCP_MAXSEG TCP_NODELAY SO_SND/RCVBUF etc..
func setSockOpt(uintFd uintptr) {
	fd := int(uintFd)

	if *enableTFO {
		err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_FASTOPEN_CONNECT, 1)
		if err != nil {
			logrus.Errorf("setsockopt TCP_FASTOPEN_CONNECT, %v", err)
		}
	}

	if *enableTCPNoDelay {
		err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1)
		if err != nil {
			logrus.Errorf("setsockopt TCP_NODELAY, %v", err)
		}
	}

	if *mss > 0 {
		err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_MAXSEG, *mss)
		if err != nil {
			logrus.Errorf("setsockopt TCP_MAXSEG, %v", err)
		}
	}
	if tcp_SO_SNDBUF > 0 {
		err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, tcp_SO_SNDBUF)
		if err != nil {
			logrus.Errorf("setsockopt SO_SNDBUF, %v", err)
		}
	}
	if tcp_SO_RCVBUF > 0 {
		err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, tcp_SO_RCVBUF)
		if err != nil {
			logrus.Errorf("setsockopt SO_RCVBUF, %v", err)
		}
	}

	return
}
