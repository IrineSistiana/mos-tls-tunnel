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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

var (
	muClientBindAddr = "127.0.0.1:50000"
	muServerBindAddr = "127.0.0.1:50001"
	muControllrAddr  = "127.0.0.1:50002"
	muDstAddr        = "127.0.0.1:50003"
)

func Test_MU(t *testing.T) {
	echo, err := newEchoServer(muDstAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer echo.close()

	wg := sync.WaitGroup{}

	//start server
	muServerTestConfig := &MUServerConfig{ServerAddr: muServerBindAddr,
		HTTPControllerAddr: muControllrAddr,
		Timeout:            time.Second * 30,
		Verbose:            true,
	}
	server, err := NewMUServer(muServerTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	wg.Add(1)
	go func() {
		fmt.Printf("server exited [%v]", server.StartTLSServerWithoutCert())
		wg.Done()
	}()
	defer server.CloseServer()

	wg.Add(1)
	go func() {
		fmt.Printf("controller exited [%v]", server.StartController())
		wg.Done()
	}()
	defer server.CloseController()

	//wait server start
	time.Sleep(500 * time.Millisecond)

	//add path to server
	path := "/qwertyu"
	cmd := MUCmd{Opt: OptAdd, ArgsBunch: []Args{Args{Path: path, Dst: muDstAddr}}}
	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf(string(b))

	resp, err := http.Post("http://"+muControllrAddr, "", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	r := new(MURes)
	err = json.Unmarshal(resBody, r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Res != ResOK {
		t.Fatal("add failed")
	}

	//start client
	cc := &ClientConfig{
		BindAddr:           muClientBindAddr,
		RemoteAddr:         muServerBindAddr,
		EnableWSS:          true,
		WSSPath:            path,
		Timeout:            time.Second * 30,
		InsecureSkipVerify: true,
		MuxMaxStream:       4,
		Verbose:            true}
	client, err := NewClient(cc)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	wg.Add(1)
	go func() {
		fmt.Printf("client exited [%v]", client.Start())
		wg.Done()
	}()

	//open tunnel
	localConn, err := net.Dial("tcp", muClientBindAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer localConn.Close()

	//test send
	localConn.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := localConn.Write(data); err != nil {
		t.Fatal(err)
	}

	//test read
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
	server.CloseController()
	server.CloseServer()
	wg.Wait()
}
