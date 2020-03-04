// +build android

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
	"syscall"

	"github.com/sirupsen/logrus"

	"golang.org/x/sys/unix"
)

func getControlFunc(conf *tcpConfig) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		if conf.vpnMode {
			if err := c.Control(sendFdToBypass); err != nil {
				return err
			}
		}
		if conf != nil {
			return c.Control(conf.setSockOpt)
		} else {
			return nil
		}
	}
}

var protectPath = "protect_path"
var unixAddr = &unix.SockaddrUnix{Name: protectPath}
var unixTimeout = &unix.Timeval{Sec: 3, Usec: 0}

func sendFdToBypass(fd uintptr) {

	socket, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		logrus.Errorf("sendFdToBypass Socket, %v", err)
		return
	}
	defer unix.Close(socket)

	unix.SetsockoptTimeval(socket, unix.SOL_SOCKET, unix.SO_RCVTIMEO, unixTimeout)
	unix.SetsockoptTimeval(socket, unix.SOL_SOCKET, unix.SO_SNDTIMEO, unixTimeout)

	err = unix.Connect(socket, unixAddr)
	if err != nil {
		logrus.Errorf("sendFdToBypass Connect, %v", err)
		return
	}

	//send fd
	if err := unix.Sendmsg(socket, nil, unix.UnixRights(int(fd)), nil, 0); err != nil {
		logrus.Errorf("sendFdToBypass Sendmsg, %v", err)
		return
	}

	//Read test ???
	unix.Read(socket, make([]byte, 1))
	return
}
