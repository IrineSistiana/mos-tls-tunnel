# mos-tls-tunnel

mos-tls-tunnel is a command line based utility that open a tls tunnel between two addresses and transfers data between them. Also support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

---

## Usage

        -b string
                Bind address (default "127.0.0.1:1080")
        -buff int
                The value of maximum socket buffer, tcp_SO_RCVBUF and tcp_SO_SNDBUF, in KB. 0 means using system default
        -cert string
                (Server only) Path to cert. If both key and cert is empty, a self signed certificate will be used
        -fallback-dns string
                use this server instead of system default to resolve host name in -b -r, must be an IP address.
        -fast-open
                (Linux kernel 4.11+ only) Enable TCP fast open
        -key string
                (Server only) Path to key. If both key and cert is empty, a self signed certificate will be used
        -max-stream int
                (Client only) The max number of multiplexed streams in one ture TCP connection (default 4)
        -mss int
                (Linux only) The value of TCP_MAXSEG. 0 means using system default
        -mux
                Enable multiplex
        -n string
                Server name. Client will use it to verify the hostname and to support virtual hosting. Server will use it to generate self signed certificate     
        -no-delay
                Enable TCP_NODELAY
        -path string
                WebSocket path (default "/")
        -r string
                Remote address
        -s    Indicate program to run as a server. If absent, run as a client
        -sv
                (Client only) Skip verify, client won't verify the server's certificate chain and host name.
        -timeout duration
                The idle timeout for connections (default 5m0s)
        -verbose
                more log
        -wss
                Enable WebSocket Secure protocol

## WebSocket Secure

mos-tls-tunnel support WebSocket Secure protocol (wss). WebSocket connections can be proxied by HTTP server such as Nginx,as well as most of CDNs that support WebSocket.

## Multiplex (Experimental)

mos-tls-tunnel support connection Multiplex (mux). It significantly reduces TCP handshake latency, at the cost of high throughput.

## Self Signed Certificate

On the server, if both `key` and `cert` is empty, a self signed certificate will be used. And the string from `n` will be certificate's `DNSName`. 

On the client, if server's certificate can't be verified. Option `sv` is required. Use it with caution.

## SIP003

mos-tls-tunnel support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

Options keys are the same as [Usage](#usage) defined

On the server, option `s` is required, for example:

        ss-server --plugin mos-tls-tunnel --plugin-opts "s"
        ss-server --plugin mos-tls-tunnel --plugin-opts "s;wss"
        ss-server --plugin mos-tls-tunnel --plugin-opts "s;cert=/path/to/your/cert;key=/path/to/your/key"
        ...

On the client:

        ss-local --plugin mos-tls-tunnel
        ss-local --plugin mos-tls-tunnel --plugin-opts "wss"
        ss-local --plugin mos-tls-tunnel --plugin-opts "sv;n=www.cloudflare.com"
        ss-local --plugin mos-tls-tunnel --plugin-opts "wss,sv;n=www.cloudflare.com,buff=32"
        ...

## Android plugin

The Android plugin project is maintained here: [mostunnel-android](https://github.com/IrineSistiana/mostunnel-android)

## Open Source Components / Libraries

* [gorilla/websocket](https://github.com/gorilla/websocket): [BSD-2-Clause](https://github.com/gorilla/websocket/blob/master/LICENSE)
* [sirupsen/logrus](https://github.com/sirupsen/logrus): [MIT](https://github.com/sirupsen/logrus/blob/master/LICENSE)
* [xtaci/smux](https://github.com/xtaci/smux): [MIT](https://github.com/xtaci/smux/blob/master/LICENSE)