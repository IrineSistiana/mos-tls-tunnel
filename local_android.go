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
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
	"unsafe"
)

func setControl(d *net.Dialer) {
	d.Control = func(network, address string, c syscall.RawConn) error {
		return c.Control(callback)
	}
}

func callback(fd uintptr) {
	path := "protect_path"

	unixConn, err := net.Dial("unix", path)
	if err != nil {
		log.Print(err)
		return
	}
	defer unixConn.Close()
	unixConn.SetDeadline(time.Now().Add(time.Second * 5))
	if _, err := unixConn.Write(uintptrToBytes(fd)); err != nil {
		log.Print(err)
		return
	}
	return
}

func uintptrToBytes(u uintptr) []byte {
	size := unsafe.Sizeof(u)
	b := make([]byte, size)
	switch size {
	case 4:
		binary.BigEndian.PutUint32(b, uint32(u))
	case 8:
		binary.BigEndian.PutUint64(b, uint64(u))
	default:
		panic(fmt.Sprintf("unknown uintptr size: %v", size))
	}
}
