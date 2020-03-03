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
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

const (
	dataSize = 1024
)

var (
	data = make([]byte, dataSize)

	clientBindAddr   = "127.0.0.1:50000"
	serverBindAddr   = "127.0.0.1:50001"
	dstAddr          = "127.0.0.1:50002"
	clientTestConfig = &ClientConfig{
		BindAddr:           clientBindAddr,
		RemoteAddr:         serverBindAddr,
		Timeout:            time.Second * 30,
		MuxMaxStream:       4,
		InsecureSkipVerify: true}
	serverTestConfig = &ServerConfig{BindAddr: serverBindAddr,
		Timeout: time.Second * 30,
		DstAddr: dstAddr,
	}
)

type dstServer struct {
	buf [1024 * 1024]byte
	l   net.Listener
}

func runDstServer(addr string, echo bool) (*dstServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	e := &dstServer{
		l: l,
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Print(err)
				return
			}
			go func() {
				for {
					i, err := conn.Read(e.buf[:])
					if err != nil {
						log.Print(err)
						return
					}
					if echo {
						_, err = conn.Write(e.buf[:i])
						if err != nil {
							log.Print(err)
							return
						}
					} else {
						ioutil.Discard.Write(e.buf[:i])
					}
				}
			}()

		}
	}()

	return e, nil
}

func (e *dstServer) close() error {
	return e.l.Close()
}

func test(sc *ServerConfig, cc *ClientConfig, t *testing.T) {
	echo, err := runDstServer(dstAddr, true)
	if err != nil {
		t.Fatal(err)
	}
	defer echo.close()

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

	localConn, err := net.Dial("tcp", clientBindAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer localConn.Close()
	localConn.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := localConn.Write(data); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, dataSize)
	_, err = localConn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf, data) {
		t.Fatal("data err")
	}

	// force to close so wg can be released
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

func Test_wss_auto_mux(t *testing.T) {
	serverTestConfig.EnableWSS = true
	serverTestConfig.WSSPath = "/"
	serverTestConfig.EnableMux = false
	clientTestConfig.EnableWSS = true
	clientTestConfig.WSSPath = "/"
	clientTestConfig.EnableMux = false
	test(serverTestConfig, clientTestConfig, t)

	clientTestConfig.EnableMux = true
	test(serverTestConfig, clientTestConfig, t)
}

func bench(sc *ServerConfig, cc *ClientConfig, b *testing.B) (conn net.Conn) {

	echo, err := runDstServer(dstAddr, false)
	if err != nil {
		b.Fatal(err)
	}
	defer echo.close()

	wg := sync.WaitGroup{}

	client, err := NewClient(clientTestConfig)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()
	wg.Add(1)
	go func() {
		fmt.Printf("client exited [%v]", client.Start())
		wg.Done()
	}()

	server, err := NewServer(serverTestConfig)
	if err != nil {
		b.Fatal(err)
	}
	wg.Add(1)
	go func() {
		fmt.Printf("server exited [%v]", server.Start())
		wg.Done()
	}()
	defer server.Close()

	time.Sleep(500 * time.Millisecond)

	localConn, err := net.Dial("tcp", clientBindAddr)
	if err != nil {
		b.Fatal(err)
	}
	defer localConn.Close()

	garbage := make([]byte, 64*1024)
	_, err = rand.Read(garbage)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	for i := 0; i < b.N; i++ {
		localConn.Write(garbage)
	}
	b.StopTimer()

	b.Logf("[%f kb/s]", float64(b.N*64)/time.Since(t).Seconds())

	// force to close so wg can be released
	client.Close()
	server.Close()
	wg.Wait()

	return
}

func Benchmark_plain(b *testing.B) {
	bench(serverTestConfig, clientTestConfig, b)
}

func Benchmark_mux(b *testing.B) {
	serverTestConfig.EnableWSS = false
	serverTestConfig.EnableMux = true
	clientTestConfig.EnableWSS = false
	clientTestConfig.EnableMux = true
	bench(serverTestConfig, clientTestConfig, b)
}

func Benchmark_wss(b *testing.B) {
	serverTestConfig.EnableWSS = true
	serverTestConfig.EnableMux = false
	serverTestConfig.WSSPath = "/"
	clientTestConfig.EnableWSS = true
	clientTestConfig.EnableMux = false
	clientTestConfig.WSSPath = "/"
	bench(serverTestConfig, clientTestConfig, b)
}

func Benchmark_wss_mux(b *testing.B) {
	serverTestConfig.EnableWSS = true
	serverTestConfig.WSSPath = "/"
	serverTestConfig.EnableMux = true
	clientTestConfig.EnableWSS = true
	clientTestConfig.WSSPath = "/"
	clientTestConfig.EnableMux = true
	bench(serverTestConfig, clientTestConfig, b)
}
