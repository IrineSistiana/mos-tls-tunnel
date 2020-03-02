# mos-tls-tunnel 多用户版服务器

---

## 实现功能

多个用户使用mtt-client的wss模式传输数据至统一的端口(如:443)，根据HTTP请求的path不同，将用户分流至相应后端(dst destination)。

`Post` json命令数据至HTTP Controller，对用户进行增删。

## 命令行

    -b string
        [Host:Port] 服务器的监听地址
    -c string
        [Host:Port] Controller的监听地址

    -mux
        启用 multiplex

    //如果不提供证书(cert和key)并且没有force-tls，mtt将监听HTTP而非HTTPS。需要格外的HTTPS反向代理才能使客户端连接。(客户端只支持HTTPS)
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

* 1: 从`args_bunch`添加用户。`args_bunch`, `path`和`dst`为必需。会覆盖重复的`path`
* 2: 按照`args_bunch`中的`path`删除用户。`args_bunch`和`path`为必需。存在的`path`会被删除。不存在的`path`会被忽略。
* 3: 重置mtt，删除所有用户数据。
* 9: 发送一个Ping，Controller回复一个OK。

**args_bunch:**

`args_bunch`中可包含多个`path`和`dst`对，但单次请求的Body不能大于2M。

**Controller json回复示例：**

    {
        "res": 1,
        "err_string":""
    }

**res:** 

* 1: 命令执行成功。
* 2: 命令执行错误，`err_string`会包含错误说明。