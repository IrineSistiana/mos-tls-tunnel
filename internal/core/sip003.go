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
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

//SIP003Args contains sip003 args
type SIP003Args struct {
	SS_REMOTE_HOST    string
	SS_REMOTE_PORT    string
	SS_LOCAL_HOST     string
	SS_LOCAL_PORT     string
	SS_PLUGIN_OPTIONS string
	VPN               bool
	TFO               bool
}

func (args *SIP003Args) GetRemoteAddr() string {
	return net.JoinHostPort(args.SS_REMOTE_HOST, args.SS_REMOTE_PORT)
}

func (args *SIP003Args) GetLocalAddr() string {
	return net.JoinHostPort(args.SS_LOCAL_HOST, args.SS_LOCAL_PORT)
}

//GetSIP003Args get sip003 args from os.Environ(), if no args, returns nil
func GetSIP003Args() (*SIP003Args, error) {
	srh, srhOk := os.LookupEnv("SS_REMOTE_HOST")
	srp, srpOk := os.LookupEnv("SS_REMOTE_PORT")
	slh, slhOk := os.LookupEnv("SS_LOCAL_HOST")
	slp, slpOk := os.LookupEnv("SS_LOCAL_PORT")
	spo, spoOk := os.LookupEnv("SS_PLUGIN_OPTIONS")

	if srhOk || srpOk || slhOk || slpOk || spoOk { // has at least one arg
		if !(srhOk && srpOk && slhOk && slpOk) { // but not has all 4 args
			return nil, ErrBrokenSIP003Args
		}
	} else {
		return nil, nil // can't find any sip003 arg
	}

	additional := flag.NewFlagSet("additional", flag.ContinueOnError)
	tfo := additional.Bool("fast-open", false, "")
	vpn := additional.Bool("V", false, "")
	additional.Parse(os.Args[1:])

	return &SIP003Args{
		SS_REMOTE_HOST:    srh,
		SS_REMOTE_PORT:    srp,
		SS_LOCAL_HOST:     slh,
		SS_LOCAL_PORT:     slp,
		SS_PLUGIN_OPTIONS: spo,

		TFO: *tfo,
		VPN: *vpn,
	}, nil
}

//FormatSSPluginOptions formats SS_PLUGIN_OPTIONS to command alike formation, `-s -a value`
func FormatSSPluginOptions(spo string) ([]string, error) {
	commandLineOption := make([]string, 0)
	op := strings.Split(spo, ";")
	for _, so := range op {
		optionPair := strings.Split(so, "=")
		switch len(optionPair) {
		case 1:
			commandLineOption = append(commandLineOption, "-"+optionPair[0])
		case 2:
			commandLineOption = append(commandLineOption, "-"+optionPair[0], optionPair[1])
		default:
			return nil, fmt.Errorf("invalid option string [%s]", so)
		}
	}

	return commandLineOption, nil
}
