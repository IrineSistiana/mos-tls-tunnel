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
	"time"

	"github.com/xtaci/smux"
)

var (
	defaultSmuxConfig = &smux.Config{
		KeepAliveInterval: 10 * time.Second,
		KeepAliveTimeout:  30 * time.Second,
		MaxFrameSize:      16 * 1024,
		MaxReceiveBuffer:  2048 * 1024,
		MaxStreamBuffer:   512 * 1024,
	}
)

const (
	defaultWSIOBufferSize   = 32 * 1024
	defaultCopyIOBufferSize = 16 * 1024
	defaultHandShakeTimeout = time.Second * 10
)

//ClientConfig is a config
type ClientConfig struct {
	BindAddr           string
	RemoteAddr         string
	EnableWSS          bool
	WSSPath            string
	ServerName         string
	InsecureSkipVerify bool
	EnableMux          bool
	MuxMaxStream       int
	Timeout            time.Duration
	EnableTFO          bool
	VpnMode            bool
	FallbackDNS        string
	Verbose            bool
}

//ServerConfig is a config
type ServerConfig struct {
	BindAddr   string
	DstAddr    string
	Key        string
	Cert       string
	EnableWSS  bool
	WSSPath    string
	ServerName string
	EnableMux  bool
	Timeout    time.Duration
	EnableTFO  bool
	Verbose    bool
}
