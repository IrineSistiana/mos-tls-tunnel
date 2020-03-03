# mos-tls-tunnel 多用户版服务器 mtt-mu-server

---

## 实现功能

mtt-mu-server可让多个用户使用mtt-client的`wss`模式传输数据至一样的服务器端口(如:443)。用户根据HTTP请求的path(`wss-path`)不同，被分流至相应的后端(`dst` destination)。

这可增加服务器的隐蔽性与安全性。因为我们不再需要暴露大量的端口给不同的用户。并且如果程序能运行在443端口，它会看起来就像是一个正常的HTTPS服务器。

每个用户都有自己唯一的 `path` 和 `dst`。

使用 HTTP 的 POST 方式将指令发送至Controller，对用户进行增删。

## 命令行

    -b string
        [Host:Port] 服务器的监听地址
    -c string
        [Host:Port] Controller的监听地址

    -mux
        启用 multiplex

    //如果不提供证书(cert和key)并且没有force-tls，
    //mtt将监听HTTP而非HTTPS。
    //需要格外的HTTPS反向代理才能使客户端连接。
    //因为客户端只支持HTTPS

    -cert string
        [Path] X509KeyPair cert file
    -key string
        [Path] X509KeyPair key file
    -force-tls
        自动生成一个证书并强制监听HTTPS 

    -timeout duration
        超时 (default 1m0s)
    -verbose
        更多Debug log 

## API

Controller 接受 HTTP POST 请求。单次请求的Body不能大于2M。

**Controller json命令格式示例：**

<details><summary><code>Add user</code></summary><br>

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

</details>

<details><summary><code>Delete user</code></summary><br>

    {
        "opt": 2,
        "args_bunch": [
            {
                "path": "/path_1"
            },
            {
                "path": "/path_2"
            }
            ...
        ]
    }

</details>

<details><summary><code>Reset server or Ping</code></summary><br>

    {
        "opt": 3
    }

    {
        "opt": 9
    }

</details>

**opt:**

* 1: Add: 从`args_bunch`添加用户。`args_bunch`, `path`和`dst`为必需。会覆盖重复的`path`。
* 2: Del: 按照`args_bunch`中的`path`删除用户。`args_bunch`和`path`为必需。存在的`path`会被删除。不存在的`path`会被忽略。
* 3: Reset: 重置mtt-mu-server，删除所有用户数据。
* 9: Ping: 发送一个Ping，Controller回复一个Pong报告当前用户数量。如果返回0可能意味着服务端已重启,需要同步用户数据。

更改或删除用户不会影响用户已建立的连接。

**args_bunch:**

`args_bunch`中可包含多个`path`和`dst`对，但单次请求的Body不能大于2M。

**Controller json回复示例：**

<details><summary><code>OK</code></summary><br>

    {
        "res": 1,
        "err_string":"",
        "current_users": 0
    }

</details>

<details><summary><code>Err</code></summary><br>

    {
        "res": 2,
        "err_string":"invalid opt",
        "current_users": 0
    }

</details>

<details><summary><code>Pong</code></summary><br>

    {
        "res": 1,
        "err_string":"",
        "current_users": 2102
    }

</details>

**res:** 

* 1: 命令执行成功。
* 2: 命令执行错误，`err_string`会包含错误说明。

**current_users:**  仅在`"opt": 9`(Ping)时有效。为当前服务器已添加的用户数。

