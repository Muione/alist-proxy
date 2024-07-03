# alist-proxy

作为 alist 下载代理，如连接中提到：[Alist 下载代理 URL](https://alist.nn.ci/zh/guide/drivers/common.html#%E4%B8%8B%E8%BD%BD%E4%BB%A3%E7%90%86-url)

## 安装
- 下载适用于你系统的最新版本 [release](https://github.com/Muione/alist-proxy/releases/latest/)

- 解压

- 运行 `alist-proxy` 会在当前目录下生成默认配置文件 `config.yaml`

- 如果没有找到对应的版本，也可以自己构建：
    - 安装 [Go](https://golang.org/doc/install) 版本需要 `1.19+`
    - clone 项目:
    ```bash
    git clone https://github.com/Muione/alist-proxy.git
    ```
    - 进入项目目录:
    ```bash
    cd alist-proxy
    ```
    - 安装依赖:
    ```bash
    go mod download
    ```
    - 构建:
    ```bash
    go build alist-proxy.go
    ```

## 配置
### `config.yaml` 文件内容如下：
```yaml
# Proxy port
port: 5243

# Use HTTPS (true/false)
https: false

# HTTPS certificate file (if https is true)
certFile: server.crt

# HTTPS key file (if https is true)
keyFile: server.key

# Alist server address
address: http://your-alist-server

# Alist server API token
token: alist-xxx
```
- `port` 监听端口，默认 `5243`
- `https` 是否使用 HTTPS，默认 `false`
- `certFile` HTTPS 证书文件，如果 `https` 为 `true` 则需要配置
- `keyFile` HTTPS 密钥文件，如果 `https` 为 `true` 则需要配置
- `address` Alist 服务器地址
- `token` Alist 服务器 API token，可以从`alist` 中 `管理-设置-其他-令牌`中获取

## 运行
- 运行 `alist-proxy` 即可

### 可以使用 `systemd` 管理程序运行
- 创建 `/etc/systemd/system/alist-proxy.service` 文件，内容如下：
```ini
[Unit]
Description=Alist Proxy Service
After=network.target

[Service]
Type=simple
WorkingDirectory=/path/to/
ExecStart=/path/to/alist-proxy
StandardOutput=append:/path/to/log.log # 将输出重定向到文件中，可以记录访问日志，防止恶意访问
StandardError=append:/path/to/err.log
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
- 修改 `/path/to/` 为你的实际路径
- 运行 `sudo systemctl daemon-reload`
- 运行 `sudo systemctl start alist-proxy`
- 运行 `sudo systemctl enable alist-proxy`
