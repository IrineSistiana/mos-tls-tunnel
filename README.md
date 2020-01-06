# mos-tls-tunnel

mos-tls-tunnel is a command line based utility that open a tls tunnel between two addresses and transfers data between them. Also support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

---

## Usage

        -b string
                Bind address (default "127.0.0.1:1080")
        -buff int
                The maximum socket buffer in KB (default 512)
        -cert string
                Path to cert, used by server mode. If both key and cert is empty, a self signed certificate will be used
        -fallback-dns string
                use this server instead of system default to resolve host name, must be an IP address.
        -fast-open
                Enable TCP fast open, only support linux with kernel version 4.11+
        -key string
                Path to key, used by server mode. If both key and cert is empty, a self signed certificate will be used
        -mss int
                TCP_MAXSEG, the maximum segment size for outgoing TCP packets, linux only
        -mux
                Enable multiplex
        -n string
                Server name, used to verify the hostname. It is also included in the client's TLS and WSS handshake to support virtual hosting unless it is an IP address.
        -no-delay
                Enable TCP_NODELAY
        -path string
                WebSocket path (default "/")
        -r string
                Remote address
        -s    Server mode
        -sv
                Skip verify, client won't verify the server's certificate chain and host name. In this mode, your connections are susceptible to man-in-the-middle attacks. Use it with caution.    
        -timeout duration
                the idle timeout for connections (default 5m0s)
        -verbose
                more log
        -wss
                Enable WebSocket Secure protocol

## WebSocket Secure

mos-tls-tunnel support WebSocket Secure protocol (wss). WebSocket connections can be proxied by HTTP server such as Nginx,as well as most of CDNs that support WebSocket.

## Multiplex (Experimental)

mos-tls-tunnel support connection Multiplex (mux). It significantly reduces TCP handshake latency, at the cost of high throughput.

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

## Self Signed Certificate

On the server, if both `key` and `cert` is empty, a self signed certificate will be used. And the string from `n` will be certificate's `DNSName`. 

On the client, if server's certificate can't be verified. Option `sv` is required. Use it with caution.

## Open Source Components / Libraries

* [gorilla/websocket](https://github.com/gorilla/websocket): [BSD-2-Clause](https://github.com/gorilla/websocket/blob/master/LICENSE)
* [sirupsen/logrus](https://github.com/sirupsen/logrus): [MIT](https://github.com/sirupsen/logrus/blob/master/LICENSE)
* [xtaci/smux](https://github.com/xtaci/smux): [MIT](https://github.com/xtaci/smux/blob/master/LICENSE)