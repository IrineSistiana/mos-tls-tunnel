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
	"io"
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
	l net.Listener
}

func runDstServer(addr string, l net.Listener, echo bool) (*dstServer, error) {
	if l == nil {
		var err error
		l, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
	}

	e := &dstServer{
		l: l,
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Printf("dummy dst server Accept, %v", err)
				return
			}
			go func() {
				defer conn.Close()
				buf := acquireIOBuf()
				defer releaseIOBuf(buf)
				for {
					i, err := conn.Read(buf)
					if err != nil {
						if err != io.EOF {
							log.Printf("dummy dst server Read, %v", err)
						}
						return
					}
					if echo {
						_, err = conn.Write(buf[:i])
						if err != nil {
							log.Printf("dummy dst server echo Write, %v", err)
							return
						}
					}
					// discard
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
	dummyConnL2C := newDummyDialerListener()
	dummyConnS2D := newDummyDialerListener()

	echo, err := runDstServer("", dummyConnS2D, true)
	if err != nil {
		t.Fatal(err)
	}
	defer echo.close()

	wg := sync.WaitGroup{}

	client, err := NewClient(clientTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	client.testDialServerRaw = dummyConnL2C.connect

	// wg.Add(1)
	// go func() {
	// 	fmt.Printf("client exited [%v]", client.Start())
	// 	wg.Done()
	// }()
	// defer client.Close()

	server, err := NewServer(serverTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	server.testDialDst = dummyConnS2D.connect
	wg.Add(1)
	go func() {
		fmt.Printf("server exited [%v]", server.ActiveAndServe(dummyConnL2C))
		wg.Done()
	}()
	defer server.Close()

	// wait server and client
	time.Sleep(500 * time.Millisecond)

	garbageSize := 16
	garbage := make([]byte, garbageSize)
	_, err = rand.Read(garbage)
	if err != nil {
		t.Fatal(err)
	}

	for g := 0; g < 1; g++ {
		wgClient := sync.WaitGroup{}
		for i := 0; i < 1; i++ {
			wgClient.Add(1)
			go func() {
				defer wgClient.Done()
				localConn, clientConn := net.Pipe()
				defer localConn.Close()
				defer clientConn.Close()
				wgClient.Add(1)
				go func() {
					defer wgClient.Done()
					err := client.ForwardConn(clientConn)
					if err != nil {
						t.Log(err)
					}
				}()
				localConn.SetDeadline(time.Now().Add(time.Second * 10))
				if _, err := localConn.Write(garbage); err != nil {
					t.Fatal(err)
				}

				buf := make([]byte, garbageSize)
				_, err = localConn.Read(buf)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(buf, garbage) {
					t.Fatal("data err")
				}
			}()
		}
		wgClient.Wait()
	}

	// force to close so wg can be released
	client.Close()
	server.Close()
	wg.Wait()
}

func Test_plain(t *testing.T) {
	test(serverTestConfig, clientTestConfig, t)
}

func Test_mux(t *testing.T) {
	serverTestConfig.EnableWSS = false
	clientTestConfig.EnableWSS = false
	serverTestConfig.EnableMux = true
	clientTestConfig.EnableMux = true
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

	dummyConnL2C := newDummyDialerListener()
	dummyConnS2D := newDummyDialerListener()

	echo, err := runDstServer("", dummyConnS2D, false)
	if err != nil {
		b.Fatal(err)
	}
	defer echo.close()

	wg := sync.WaitGroup{}

	client, err := NewClient(clientTestConfig)
	if err != nil {
		b.Fatal(err)
	}
	client.testDialServerRaw = dummyConnL2C.connect

	// server
	server, err := NewServer(serverTestConfig)
	if err != nil {
		b.Fatal(err)
	}
	server.testDialDst = dummyConnS2D.connect
	wg.Add(1)
	go func() {
		fmt.Printf("server exited [%v]", server.ActiveAndServe(dummyConnL2C))
		wg.Done()
	}()
	defer server.Close()

	// wait server and client
	time.Sleep(500 * time.Millisecond)

	localConn, clientConn := net.Pipe()
	defer localConn.Close()
	defer clientConn.Close()
	go func() {
		err := client.ForwardConn(clientConn)
		if err != nil {
			log.Printf("client forward: %v", err)
		}
	}()

	garbage := make([]byte, 64*1024)
	_, err = rand.Read(garbage)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	c := 0
	for i := 0; i < b.N; i++ {
		n, err := localConn.Write(garbage)
		if err != nil {
			b.Fatalf("write data @ %d %.2f%%, %v", c, float64(c)/float64(b.N*64*1024), err)
		}
		c = c + n
	}
	b.StopTimer()
	b.Logf("[%dM %.2f Mb/s]", c/1024/1024, float64(c/1024/1024)/time.Since(t).Seconds())

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
