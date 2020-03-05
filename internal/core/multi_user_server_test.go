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
	"encoding/json"
	"errors"
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
	echo, err := runDstServer(muDstAddr, true)
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
	muServer, err := NewMUServer(muServerTestConfig)
	if err != nil {
		t.Fatal(err)
	}
	wg.Add(1)
	go func() {
		fmt.Printf("server exited [%v]", muServer.StartServer())
		wg.Done()
	}()
	defer muServer.CloseServer()

	wg.Add(1)
	go func() {
		fmt.Printf("controller exited [%v]", muServer.StartController())
		wg.Done()
	}()
	defer muServer.CloseController()

	//wait server start
	time.Sleep(500 * time.Millisecond)

	sendCmd := func(opt int, path, dst string) (int, error) {
		cmd := MUCmd{Opt: opt, ArgsBunch: []Args{Args{Path: path, Dst: dst}}}
		b, err := json.Marshal(cmd)
		if err != nil {
			return 0, err
		}

		resp, err := http.Post("http://"+muControllrAddr, "", bytes.NewReader(b))
		if err != nil {
			return 0, err
		}

		resBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		r := new(MURes)
		err = json.Unmarshal(resBody, r)
		if err != nil {
			return 0, err
		}
		if r.Res != ResOK {
			return 0, fmt.Errorf("add user failed, server faied with err [%s]", r.ErrString)
		}
		return r.CurrentUsers, nil
	}

	testClient := func(c *ClientConfig) error {
		wg := sync.WaitGroup{}

		client, err := NewClient(c)
		if err != nil {
			return fmt.Errorf("NewClient: %v", err)
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
			return fmt.Errorf("connecting to client: %v", err)
		}
		defer localConn.Close()

		//test send
		garbageSize := 4096
		garbage := make([]byte, garbageSize)
		_, err = rand.Read(garbage)
		if err != nil {
			t.Fatal(err)
		}

		localConn.SetWriteDeadline(time.Now().Add(time.Second))
		if _, err := localConn.Write(garbage); err != nil {
			return fmt.Errorf("write to client: %v", err)
		}

		//test read
		buf := make([]byte, garbageSize)
		_, err = localConn.Read(buf)
		if err != nil {
			return fmt.Errorf("read from client: %v", err)
		}
		if !bytes.Equal(buf, garbage) {
			t.Fatal("data err")
		}

		client.Close()
		wg.Wait()
		return nil
	}

	//add first user
	_, err = sendCmd(OptAdd, "/qwertyu", muDstAddr)
	if err != nil {
		t.Fatal(err)
	}

	//start client
	cc := &ClientConfig{
		BindAddr:           muClientBindAddr,
		RemoteAddr:         muServerBindAddr,
		EnableWSS:          true,
		WSSPath:            "/qwertyu",
		Timeout:            time.Second * 30,
		InsecureSkipVerify: true,
		MuxMaxStream:       4,
		Verbose:            false,
	}

	//test tunnel
	err = testClient(cc)
	if err != nil {
		t.Fatal(err)
	}

	//test ping
	n, err := sendCmd(OptPing, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal(errors.New("test ping failed"))
	}

	//del first user
	_, err = sendCmd(OptDel, "/qwertyu", muDstAddr)
	if err != nil {
		t.Fatal(err)
	}

	err = testClient(cc)
	if err == nil {
		t.Fatal(errors.New("err is expected"))
	}

	// force to close so wg can be released
	muServer.CloseController()
	muServer.CloseServer()
	wg.Wait()
}
