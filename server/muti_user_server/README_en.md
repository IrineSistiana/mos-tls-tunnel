# mos-tls-tunnel multi-user server (mtt-mu-server)

---

## What can multi-user server do

mtt-mu-server allows multiple users to use the `wss` mode of mtt-client to transfer data on the same server port (eg: 443). Users are offloaded to the corresponding backend (`dst` destination) according to the path (`wss-path`) of their HTTP request.

This can increase the concealment and security of the server. Because we no longer need to expose a large number of ports to different users. And if mtt-mu-server can run on port 443, it will look like a normal HTTPS server.

Each user has their own unique `path` and `dst`.

Use HTTP's POST method to send commands to the Controller to add or delete users.

## Command Line

    -b string
        [Host:Port] Server address, e.g. '127.0.0.1:1080'
    -c string
        [Host:Port]  Controller address

    -mux
        Enable multiplex

    // If no certificate (cert and key) is provided and there is no force-tls, 
    // mtt will listen on HTTP instead of HTTPS.
    // An extra HTTPS reverse proxy is required for clients to connect. 
    // Because the client only supports HTTPS

    -cert string
        [Path] X509KeyPair cert file
    -key string
        [Path] X509KeyPair key file
    -force-tls
        automatically generates a certificate for listening on HTTPS

    -timeout duration
        The idle timeout for connections (default 1m0s)
    -verbose
        more log

## API

The Controller accepts HTTP POST requests. The body of a single request cannot be greater than 2M.

**Controller json command format example:**

    {
        "opt": 1,
        "args_bunch": [
            {
                "path": "/path_1",
                "dst": "127.0.0.1:10001"
            },
            {
                "path": "/path_2",
                "dst": "127.0.0.1:10002"
            }

            ...
        ]
    }

**opt:**

* 1: Add users from `args_bunch`. `args_bunch`,`path` and `dst` are required. Repeated `path` will be overwrited.
* 2: Delete the user by `path` in `args_bunch`. `args_bunch` and `path` are required. The existing `path` will be deleted. Non-existent `path`s are ignored.
* 3: Reset server, delete all users.
* 9: Send a Ping, Controller responds with an OK.

**args_bunch:**

`args_bunch` can contain multiple `path` and `dst` pairs, but the body of a single request cannot be greater than 2M.

**Controller json response example:**

    {
        "res": 1,
        "err_string":""
    }

**res:**

* 1: The command was executed successfully.
* 2: An error occurred, `err_string` will contain error description.