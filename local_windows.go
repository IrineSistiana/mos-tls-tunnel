// +build windows

// Copyright (c) 2019 IrineSistiana
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

func getControlFunc() func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		return c.Control(setSockOpt)
	}
}

//TCP_MAXSEG TCP_NODELAY SO_SND/RCVBUF etc..
func setSockOpt(uintptrFd uintptr) {
	fd := windows.Handle(uintptrFd)
	var err error

	if *noDelay {
		err = windows.SetsockoptInt(fd, windows.IPPROTO_TCP, windows.TCP_NODELAY, 1)
		if err != nil {
			logrus.Print(err)
		}
	}

	// can't set TCP_MAXSEG on windows

	// if *mss > 0 {
	// 	windows.SetsockoptInt(fd, windows.IPPROTO_TCP, windows.TCP_MAXSEG, *mss)
	// }

	err = windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_SNDBUF, *buffSize*1024)
	if err != nil {
		logrus.Print(err)
	}
	err = windows.SetsockoptInt(fd, windows.SOL_SOCKET, windows.SO_RCVBUF, *buffSize*1024)
	if err != nil {
		logrus.Print(err)
	}

	return
}
