# mos-tls-tunnel

mos-tls-tunnel is a command line based utility that open a tls tunnel between two addresses and transfers data between them. Also support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

---

## Usage

        -V    vpn mode, used in android system only
        -b string
                bind address (default "127.0.0.1:1080")
        -buff int
                size of io buffer for each connection (kb) (default 32)
        -cert string
                path to cert, used by server mode, if both key and cert is empty, a self signed certificate will be used
        -fast-open
                not support yet, reserved
        -key string
                path to key, used by server mode, if both key and cert is empty, a self signed certificate will be used
        -n string
                server name, used to verify the hostname. it is also included in the client's handshake to support virtual hosting unless it is an IP address.
        -path string
                WebSocket path (default "/")
        -r string
                remote address
        -s    server mode
        -sv
                skip verifies the server's certificate chain and host name. use it with caution.
        -timeout duration
                read and write timeout deadline(sec) (default 5m0s)
        -wss
                using WebSocket Secure protocol

## WebSocket Secure

mos-tls-tunnel support WebSocket Secure protocol (wss). WebSocket connections can be proxied by HTTP server such as Nginx,as well as most of CDNs that support WebSocket.

## SIP003

mos-tls-tunnel support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

Options keys are the same as [Usage](#usage) defined

On the server, option `s` is required, for example:

        ss-server --plugin mos-tls-tunnel --plugin-opts "s"
        ss-server --plugin mos-tls-tunnel --plugin-opts "s;wss"
        ss-server --plugin mos-tls-tunnel --plugin-opts "s;cert=/path/to/your/cert;key=/path/to/your/key"
        ...

On the client, option `n` is required:

        ss-local --plugin mos-tls-tunnel --plugin-opts "wss;n=www.cloudflare.com"
        ss-local --plugin mos-tls-tunnel --plugin-opts "sv;n=www.cloudflare.com"
        ...

## Self Signed Certificate

On the server, if both `key` and `cert` is empty, a self signed certificate will be used. And the string from `n` will be certificate's `DNSName`. 

On the client, if server's certificate can't be verified. Option `sv` is required. Use it with caution.