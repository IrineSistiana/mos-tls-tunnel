// +build android

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
	"log"
	"syscall"

	"golang.org/x/sys/unix"
)

func getControlFunc() func(network, address string, c syscall.RawConn) error {
	if *vpnMode {
		return func(network, address string, c syscall.RawConn) error {
			return c.Control(callback)
		}
	} else {
		return nil
	}
}

var protectPath = "protect_path"
var unixAddr = &unix.SockaddrUnix{Name: protectPath}
var unixTimeout = &unix.Timeval{Sec: 3, Usec: 0}

func callback(fd uintptr) {

	socket, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		log.Print(err)
		return
	}
	defer unix.Close(socket)

	unix.SetsockoptTimeval(socket, unix.SOL_SOCKET, unix.SO_RCVTIMEO, unixTimeout)
	unix.SetsockoptTimeval(socket, unix.SOL_SOCKET, unix.SO_SNDTIMEO, unixTimeout)

	err = unix.Connect(socket, unixAddr)
	if err != nil {
		log.Print(err)
		return
	}

	//send fd
	if err := unix.Sendmsg(socket, nil, unix.UnixRights(int(fd)), nil, 0); err != nil {
		log.Print(err)
		return
	}

	//Read test ???
	unix.Read(socket, make([]byte, 1))
	return
}
