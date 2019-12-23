# mos-tls-tunnel

mos-tls-tunnel is a command line based utility that open a tls tunnel between two addresses and transfers data between them. Also support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

---

## Usage

        -b string
                bind address (default "127.0.0.1:1080")
        -buff int
                size of io buffer for each connection (kb) (default 32)
        -cert string
                path to cert, used by server mode, if both key and cert is empty, a self signed certificate will be used
        -key string
                path to key, used by server mode, if both key and cert is empty, a self signed certificate will be used
        -n string
                server name, used to verify the hostname. it is also included in the client's handshake to support virtual hosting unless it is an IP address.
        -r string
                remote address
        -s    server mode
        -sv
                skip verifies the server's certificate chain and host name. use it with caution.
        -timeout int
                read and write timeout deadline(sec) (default 300)
        -verbose
                moooore log

## SIP003

mos-tls-tunnel support shadowsocks [SIP003](https://shadowsocks.org/en/spec/Plugin.html)

On the server, for example:

        ss-server --plugin mos-tls-tunnel --plugin-opts "s"
        ss-server --plugin mos-tls-tunnel --plugin-opts "s;buff=4"
        ss-server --plugin mos-tls-tunnel --plugin-opts "s;cert=/path/to/your/cert;key=/path/to/your/key"

On the client:

        ss-local --plugin mos-tls-tunnel
        ss-local --plugin mos-tls-tunnel --plugin-opts "sv;n=www.cloudflare.com"

## Suggestions

mos-tls-tunnel is simpler, no dependencies.

Alternative: `nginx`