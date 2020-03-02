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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	maxControlBodySize = 1024 * 1024 * 2
)

//MUCmd is a control command
type MUCmd struct {
	Opt int `json:"opt,omitempty"`

	ArgsBunch []Args `json:"args_bunch,omitempty"`
}

type Args struct {
	Path string `json:"path,omitempty"`
	Dst  string `json:"dst,omitempty"`
}

//MUCmd opt id
const (
	OptAdd   = 1
	OptDel   = 2
	OptReset = 3
	OptPing  = 9
)

//MURes is a update result
type MURes struct {
	Res       int    `json:"res,omitempty"`
	ErrString string `json:"err_string,omitempty"`
}

//MURes res id
const (
	ResOK  = 1
	ResErr = 2
)

//MUServerConfig multi-user server config
type MUServerConfig struct {
	ServerAddr         string
	HTTPControllerAddr string

	EnableMux bool
	Timeout   time.Duration
	Verbose   bool
}

//MUServer is a multi-user server
type MUServer struct {
	mux        *mux
	server     http.Server
	controller http.Server
	logger     *logrus.Logger
}

//NewMUServer init a multi-user server
func NewMUServer(conf *MUServerConfig) (*MUServer, error) {

	mus := new(MUServer)

	//logger
	mus.logger = logrus.New()
	if conf.Verbose {
		mus.logger.SetLevel(logrus.InfoLevel)
	} else {
		mus.logger.SetLevel(logrus.ErrorLevel)
	}

	mus.mux = newMux(conf.EnableMux, conf.Timeout, mus.logger)

	mus.server = http.Server{Addr: conf.ServerAddr, Handler: mus.mux}
	mus.controller = http.Server{Addr: conf.HTTPControllerAddr, Handler: mus}

	return mus, nil
}

//StartTLSServerWithoutCert starts server with self signed cert
func (mus *MUServer) StartTLSServerWithoutCert() error {
	tlsConf := new(tls.Config)
	cers, err := generateCertificate("")
	if err != nil {
		return fmt.Errorf("generate certificate: %v", err)
	}
	logrus.Print("WARNING: you are using a self-signed certificate")
	tlsConf.Certificates = cers
	mus.server.TLSConfig = tlsConf
	return mus.server.ListenAndServeTLS("", "")
}

//StartTLSServer starts the server in tls mode
func (mus *MUServer) StartTLSServer(certFile, keyFile string) error {
	return mus.server.ListenAndServeTLS(certFile, keyFile)
}

//StartHTTPServer starts the server in http mode
func (mus *MUServer) StartHTTPServer() error {
	return mus.server.ListenAndServe()
}

//StartController starts the controller of the server
func (mus *MUServer) StartController() error {
	return mus.controller.ListenAndServe()
}

func (mus *MUServer) CloseController() error {
	return mus.controller.Close()
}

func (mus *MUServer) CloseServer() error {
	return mus.server.Close()
}

func (mus *MUServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		mus.logger.Warnf("read command body from %s failed, method is [%s]", r.RemoteAddr, r.Method)
		return
	}
	var buf bytes.Buffer
	lr := io.LimitReader(r.Body, maxControlBodySize)
	n, err := buf.ReadFrom(lr)
	if n == maxControlBodySize {
		mus.logger.Warnf("read command body from %s failed, body is larger than 2m", r.RemoteAddr)
		return
	}
	if err != nil {
		mus.logger.Warnf("read command body from %s failed, %v", r.RemoteAddr, err)
		return
	}
	muCmd := new(MUCmd)
	if err := json.Unmarshal(buf.Bytes(), muCmd); err != nil {
		mus.logger.Warnf("unmarshal command body from %s failed, %v", r.RemoteAddr, err)
		return
	}

	switch muCmd.Opt {
	case OptAdd:
		if len(muCmd.ArgsBunch) == 0 {
			sendMURes(w, ResErr, "empty args")
			return
		}
		mus.mux.add(muCmd.ArgsBunch)
		sendMURes(w, ResOK, "")
	case OptDel:
		if len(muCmd.ArgsBunch) == 0 {
			sendMURes(w, ResErr, "empty args")
			return
		}
		mus.mux.del(muCmd.ArgsBunch)
		sendMURes(w, ResOK, "")
	case OptReset:
		mus.mux.reset()
		sendMURes(w, ResOK, "")
	case OptPing:
		sendMURes(w, ResOK, "")
	default:
		mus.logger.Warnf("invalid Opt from %s , %d", r.RemoteAddr, muCmd.Opt)
		sendMURes(w, ResErr, "invalid Opt")
	}
}

func sendMURes(w http.ResponseWriter, res int, errStr string) error {
	data, err := generateMURes(res, errStr)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func generateMURes(res int, errStr string) ([]byte, error) {
	muRes := MURes{
		Res:       res,
		ErrString: errStr,
	}
	b, err := json.Marshal(muRes)
	if err != nil {
		return nil, err
	}
	return b, nil
}
