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
	"bytes"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

var (
	dataSize = 1024 * 128
	data     = make([]byte, dataSize)

	clientBindAddr   = "127.0.0.1:50000"
	serverBindAddr   = "127.0.0.1:50001"
	dstAddr          = "127.0.0.1:50002"
	clientTestConfig = &ClientConfig{
		BindAddr:           clientBindAddr,
		RemoteAddr:         serverBindAddr,
		Timeout:            time.Second * 30,
		InsecureSkipVerify: true}
	serverTestConfig = &ServerConfig{BindAddr: serverBindAddr,
		Timeout: time.Second * 30,
		DstAddr: dstAddr,
	}
)

func test(sc *ServerConfig, cc *ClientConfig, t *testing.T) {
	wg := sync.WaitGroup{}

	client, err := NewClient(clientTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	wg.Add(1)
	go func() {
		fmt.Printf("client exited [%v]", client.Start())
		wg.Done()
	}()

	server, err := NewServer(serverTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	wg.Add(1)
	go func() {
		fmt.Printf("server exited [%v]", server.Start())
		wg.Done()
	}()
	defer server.Close()

	time.Sleep(500 * time.Millisecond)
	l, err := net.Listen("tcp", dstAddr)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		localConn, err := net.Dial("tcp", clientBindAddr)
		if err != nil {
			t.Fatal(err)
		}
		localConn.SetWriteDeadline(time.Now().Add(time.Second))
		if _, err := localConn.Write(data); err != nil {
			t.Fatal(err)
		}
		localConn.Close()
	}()
	go func() {
		time.Sleep(time.Second * 2)
		l.Close()
	}()

	dstConn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	dstConn.SetWriteDeadline(time.Now().Add(time.Second))
	buf := make([]byte, dataSize)
	i := 0
	for i < dataSize {
		n, err := dstConn.Read(buf[i:])
		i = i + n
		if err != nil {
			break
		}
	}
	if !bytes.Equal(buf, data) {
		t.Fatal("data err")
	}

	client.Close()
	server.Close()
	wg.Wait()
}

func Test_plain(t *testing.T) {
	test(serverTestConfig, clientTestConfig, t)
}

func Test_wss(t *testing.T) {
	serverTestConfig.EnableWSS = true
	serverTestConfig.WSSPath = "/"
	clientTestConfig.EnableWSS = true
	clientTestConfig.WSSPath = "/"
	test(serverTestConfig, clientTestConfig, t)
}

func Test_wss_mux(t *testing.T) {
	serverTestConfig.EnableWSS = true
	serverTestConfig.WSSPath = "/"
	serverTestConfig.EnableMux = true
	clientTestConfig.EnableWSS = true
	clientTestConfig.WSSPath = "/"
	clientTestConfig.EnableMux = true
	test(serverTestConfig, clientTestConfig, t)
}
