# mos-tls-tunnel

mos-tls-tunnel is a command line based utility that open a tls tunnel between two addresses and transfers data between them. Also support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

---

## Usage

client ---> |mtt-client| ---> |mtt-server| ---> destination

`mtt-client`:

        -b string
                [Host:Port] Bind address, e.g. '127.0.0.1:1080'
        -fallback-dns string
                [IP:Port] Use this server instead of system default to resolve host name in -b -r, must be an IP address.
        -fast-open
                (Linux kernel 4.11+ only) Enable TCP fast open
        -mux
                Enable multiplex
        -mux-max-stream int
                The max number of multiplexed streams in one ture TCP connection (default 4)
        -n string
                Server name. Use to verify the hostname and to support virtual hosting.
        -s string
                [Host:Port] Server address
        -sv
                Skip verify. Client won't verify the server's certificate chain and host name.
        -timeout duration
                The idle timeout for connections (default 5m0s)
        -verbose
                more log
        -wss
                Enable WebSocket Secure protocol
        -wss-path string
                WebSocket path (default "/")

`mtt-server`:

        -b string
                [Host:Port] Bind address, e.g. '127.0.0.1:1080'
        -cert string
                [Path] X509KeyPair cert file
        -fast-open
                (Linux kernel 4.11+ only) Enable TCP fast open
        -key string
                [Path] X509KeyPair key file
        -mux
                Enable multiplex
        -n string
                Server name. Use to generate self signed certificate DNSName
        -d string
                [Host:Port] Destination address
        -timeout duration
                The idle timeout for connections (default 5m0s)
        -verbose
                more log
        -wss
                Enable WebSocket Secure protocol
        -wss-path string
                WebSocket path (default "/")


 **Note**: These options need to be consistent across client and server: `mux`,`wss`,`wss-path`

## WebSocket Secure

mos-tls-tunnel support WebSocket Secure protocol (`wss`). WebSocket connections can be proxied by HTTP server such as Apache, as well as most of CDNs that support WebSocket.

## Multiplex (Experimental)

mos-tls-tunnel support connection Multiplex (`mux`). It significantly reduces TCP handshake latency, at the cost of high throughput.

## Self Signed Certificate

On the server, if both `key` and `cert` is empty, a self signed certificate will be used. And the string from `n` will be certificate's `DNSName`. 

On the client, if server's certificate can't be verified. Option `sv` is required. Use it with caution.

## SIP003

mos-tls-tunnel support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html).Options keys are the same as [Usage](#usage) defined.

For example:

**Shadowsocks over TLS**:

        ss-server -c config.json --plugin mtt-server 
        ss-local -c config.json --plugin mtt-client --plugin-opts "sv"

**Shadowsocks over WebSocket Secure(wss)**:

        ss-server -c config.json --plugin mtt-server --plugin-opts "wss"
        ss-local -c config.json --plugin mtt-client --plugin-opts "wss,sv"



## Android plugin

The Android plugin project is maintained here: [mostunnel-android](https://github.com/IrineSistiana/mostunnel-android)

## Open Source Components / Libraries

* [gorilla/websocket](https://github.com/gorilla/websocket): [BSD-2-Clause](https://github.com/gorilla/websocket/blob/master/LICENSE)
* [sirupsen/logrus](https://github.com/sirupsen/logrus): [MIT](https://github.com/sirupsen/logrus/blob/master/LICENSE)
* [xtaci/smux](https://github.com/xtaci/smux): [MIT](https://github.com/xtaci/smux/blob/master/LICENSE)