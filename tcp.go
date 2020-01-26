package main

type tcpConfig struct {
	tfo     bool
	noDelay bool
	mss     int
	sndBuf  int
	rcvBuf  int
}
