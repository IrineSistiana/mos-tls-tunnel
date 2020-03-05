# mos-tls-tunnel

mos-tls-tunnel is a command line based utility that open a tls tunnel between two addresses and transfers data between them. Also support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html) and [multi-user server](#mtt-server-multi-user-version-mtt-mu-server).

---

- [mos-tls-tunnel](#mos-tls-tunnel)
  - [Usage](#usage)
    - [mtt-client](#mtt-client)
    - [mtt-server](#mtt-server)
    - [mtt-mu-server](#mtt-mu-server)
  - [WebSocket Secure](#websocket-secure)
  - [Multiplex (Experimental)](#multiplex-experimental)
  - [Self Signed Certificate](#self-signed-certificate)
  - [Shadowsocks Plugin (SIP003)](#shadowsocks-plugin-sip003)
    - [Example Command](#example-command)
    - [Recommended Shadowsocks server and client](#recommended-shadowsocks-server-and-client)
    - [Android plugin](#android-plugin)
  - [mtt-server Multi-user Version (mtt-mu-server)](#mtt-server-multi-user-version-mtt-mu-server)
  - [Build from Source](#build-from-source)
  - [Open Source Components / Libraries](#open-source-components--libraries)

## Usage

client ---> |mtt-client| ---> |mtt-server| ---> destination

 **Note**: In order for the client to connect to the server normally, the following options must be consistent between the client and the server. In other words, if the server has this option, the client must also have this option, and vice versa.

* if server enabled `wss`: `wss` and `wss-path` must be consistent.
* if server NOT enabled `wss`: `wss` and `mux` must be consistent.

### mtt-client

    -b string
        [Host:Port] Bind address, e.g. '127.0.0.1:1080'
    -s string
        [Host:Port] Server address

    -wss
        Enable WebSocket Secure protocol
    -wss-path string
        WebSocket path (default "/")
    -mux
        Enable multiplex
    -mux-max-stream int
        The max number of multiplexed streams in one ture TCP connection, 1 - 16 (default 4)

<details><summary><code>Geek options</code></summary><br>

    -sv
        Skip verify. Client won't verify the server's certificate chain and host name.
    -fast-open
        (Linux kernel 4.11+ only) Enable TCP fast open
    -n string
        Server name. Use to verify the hostname and to support virtual hosting.

    -timeout duration
        The idle timeout for connections (default 5m0s)
    -fallback-dns string
        [IP:Port] Use this server instead of system default to resolve host name in -b -r, must be an IP address.
    -verbose
        more log

</details>

### mtt-server

    -b string
        [Host:Port] or [Path](if bind-unix) Server bind address, e.g. '127.0.0.1:1080', '/run/mmt-server', '@mmt-server'
    -d string
        [Host:Port] Destination address

    -wss
        Enable WebSocket Secure protocol
    -wss-path string
        WebSocket path (default "/")
    -mux
        Enable multiplex

    -cert string
        [Path] X509KeyPair cert file
    -key string
        [Path] X509KeyPair key file

<details><summary><code>Geek options</code></summary><br>

    -bind-unix 
        Bind on unix socket instead of TCP socket. 
    -fast-open
        (Linux kernel 4.11+ only) Enable TCP fast open
    -disable-tls
        disable TLS. An extra TLS proxy is required, such as Nginx SSL Stream Module
    -n string
        Server name. Use to generate self signed certificate DNSName

    -timeout duration
        The idle timeout for connections (default 5m0s)
    -verbose
        more log

</details>

### mtt-mu-server

See [here](#mtt-server-multi-user-version-mtt-mu-server)

## WebSocket Secure

mos-tls-tunnel support WebSocket Secure protocol (`wss`). WebSocket connections can be proxied by HTTP server such as Apache, as well as most of CDNs that support WebSocket.

`wss-path` will be the path of HTTP request.

## Multiplex (Experimental)

mos-tls-tunnel support connection Multiplex (`mux`). It significantly reduces handshake latency, at the cost of high throughput.

Client can set `mux-max-stream` to control the maximum number of data streams in one TCP connection. The value should be between 1 and 16.

if `wss` is enabled, server can automatically detect whether client enable `mux` or not. But you can still use the `mux` to force the server to enable multiplex if auto-detection fails.

## Self Signed Certificate

On the server, if both `key` and `cert` is empty, a self signed certificate will be used. And the string from `n` will be certificate's hostname. **This self signed certificate CANNOT be verified.**

On the client, if server's certificate can't be verified. You can enable `sv` to skip the verification. **Enable this option only if you know what you are doing. Use it with caution.**

We recommend that you use a valid certificate all the time. A free and valid certificate can be easily obtained here. [Let's Encrypt](https://letsencrypt.org/)

## Shadowsocks Plugin (SIP003)

mos-tls-tunnel support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html). Options keys are the same as [Usage](#usage) defined. You don't have to set client and server address: `b`,`d`,`s`, shadowsocks will set those automatically. 

### Example Command 

Below are example commands with [shadowsocks-libev](https://github.com/shadowsocks/shadowsocks-libev).

**Shadowsocks over TLS**

    ss-server -c config.json --plugin mtt-server --plugin-opts "key=/path/to/your/key;cert=/path/to/your/cert"
    ss-local -c config.json --plugin mtt-client --plugin-opts "n=your.server.hostname"

**Shadowsocks over WebSocket Secure(wss)**

    ss-server -c config.json --plugin mtt-server --plugin-opts "wss,key=/path/to/your/key;cert=/path/to/your/cert"
    ss-local -c config.json --plugin mtt-client --plugin-opts "wss;n=your.server.hostname"

### Recommended Shadowsocks server and client

* [shadowsocks-libev](https://github.com/shadowsocks/shadowsocks-libev)
* [shadowsocks-windows](https://github.com/shadowsocks/shadowsocks-windows)
* [shadowsocks-android](https://github.com/shadowsocks/shadowsocks-android)

### Android plugin

The Android plugin project is maintained here: [mostunnel-android](https://github.com/IrineSistiana/mostunnel-android). This is a plugin of [shadowsocks-android](https://github.com/shadowsocks/shadowsocks-android).

## mtt-server Multi-user Version (mtt-mu-server)

mtt-mu-server allows multiple users to use the `wss` mode of mtt-client to transfer data on the same server port (eg: 443). Users are offloaded to the corresponding backend (`dst` destination) according to the path (`wss-path`) of their HTTP request.

This can increase the concealment and security of the server. Because we no longer need to expose a large number of ports to different users. And if mtt-mu-server can run on port 443, it will look like a normal HTTPS server.

API is very simple: Use HTTP's POST method to send commands to the Controller to add or delete as many users as you want.

For more, see [here](cmd/mtt-mu-server).

## Build from Source

In general, you need the following build dependencies:

* golang-go 
* git

You might build mos-tls-tunnel like this:

    # get source
    go get -d -u github.com/IrineSistiana/mos-tls-tunnel/cmd/mtt-client
    go get -d -u github.com/IrineSistiana/mos-tls-tunnel/cmd/mtt-server
    go get -d -u github.com/IrineSistiana/mos-tls-tunnel/cmd/mtt-mu-server

    # start building
    go build -o ./ github.com/IrineSistiana/mos-tls-tunnel/cmd/mtt-client
    go build -o ./ github.com/IrineSistiana/mos-tls-tunnel/cmd/mtt-server
    go build -o ./ github.com/IrineSistiana/mos-tls-tunnel/cmd/mtt-mu-server
    
## Open Source Components / Libraries

* [gorilla/websocket](https://github.com/gorilla/websocket): [BSD-2-Clause](https://github.com/gorilla/websocket/blob/master/LICENSE)
* [sirupsen/logrus](https://github.com/sirupsen/logrus): [MIT](https://github.com/sirupsen/logrus/blob/master/LICENSE)
* [xtaci/smux](https://github.com/xtaci/smux): [MIT](https://github.com/xtaci/smux/blob/master/LICENSE)