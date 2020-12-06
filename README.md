# ngredt
内网穿透,内网tcp穿透,http/https代理

```
服务端配置：
ngredt_sever.conf

{
  "sgn_port": 1001, #本机信令端口
  "conn_port": 1002, #本机连接池端口
  "port_map": { # 端口映射，praviteKey-->port
    "client_key1": 8888,# 将key1分配给8888端口，供公网访问
    "client_key2": 8001 # 将key2分配给8001端口，供公网访问
  }
}

服务端执行 ./ngreds ngredt_sever.conf


客户端配置：
ngredt_client.conf

{
  "sev_host": "1721.123.12312.1241", #这里是远程公网服务的地址
  "sgn_port": 1001, #远程服务端的信令端口
  "conn_port": 1002, #远程服务端的连接池端口
  "loc_addr": "127.0.0.1:8080", # 本机服务地址
  "private_key": "client_key1" # 本机的private_key
}

客户端启动执行：./ngredt ngredt_client.conf

启动后访问1721.123.12312.1241:8888 ，即可访问到本机127.0.0.1:8080端口
```
