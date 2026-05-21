# Go Proxy IPv6 Pool

Go Proxy IPv6 Pool 是一个随机 IPv6 出口代理服务，支持 HTTP 代理和 SOCKS5 代理。

当前版本支持：

- 动态随机 IPv6 出口端口
- 固定端口绑定固定 IPv6 出口
- HTTP 代理账号密码认证
- SOCKS5 代理账号密码认证
- 固定端口 IPv6 分配持久化
- 保留旧版 `--port` / `--cidr` 启动方式

## 工作方式

程序需要一个可用的 IPv6 CIDR，例如：

```text
2001:399:8205:ae00::/64
```

动态端口会在每次出站连接时从 CIDR 中随机生成一个 IPv6，并把它作为本地出口源地址。

固定端口会在首次启动时为每个端口随机分配一个 IPv6，并写入状态文件。之后服务重启时，同一个固定端口会继续使用同一个 IPv6。

## 配置文件

推荐使用配置文件启动。可以复制示例配置：

```bash
cp config.example.yaml config.yaml
```

示例：

```yaml
cidr: "2001:399:8205:ae00::/64"
state_file: "state.json"
verbose: false

auth:
  username: "proxy_user"
  password: "proxy_password"

dynamic:
  http_port: 52122
  socks5_port: 52123

fixed:
  http_ports:
    - 52133
    - 52134
  socks5_ports:
    - 52135
```

字段说明：

- `cidr`：IPv6 网段，必填。
- `state_file`：固定端口 IPv6 映射的持久化文件，默认 `state.json`。
- `verbose`：是否打印更详细的 HTTP 代理日志。
- `auth.username` / `auth.password`：代理认证账号密码。两者都为空表示不启用认证；只配置其中一个会启动失败。
- `dynamic.http_port`：动态 HTTP 代理端口。
- `dynamic.socks5_port`：动态 SOCKS5 代理端口。
- `fixed.http_ports`：固定 IPv6 的 HTTP 代理端口列表。
- `fixed.socks5_ports`：固定 IPv6 的 SOCKS5 代理端口列表。

## 启动

使用默认配置文件 `config.yaml`：

```bash
go run .
```

指定配置文件：

```bash
go run . --config ./config.yaml
```

如果当前目录没有 `config.yaml`，也可以继续使用旧版方式启动动态端口：

```bash
go run . --port 52122 --cidr "2001:399:8205:ae00::/64"
```

旧版方式会启动：

- HTTP 动态代理：`52122`
- SOCKS5 动态代理：`52123`

旧版方式不会配置账号密码，也不会配置固定端口。

## 使用动态代理

假设配置如下：

```yaml
auth:
  username: "proxy_user"
  password: "proxy_password"

dynamic:
  http_port: 52122
  socks5_port: 52123
```

HTTP 代理：

```bash
curl -x http://proxy_user:proxy_password@服务器IP:52122 http://6.ipw.cn/
```

SOCKS5 代理：

```bash
curl -x socks5://proxy_user:proxy_password@服务器IP:52123 http://6.ipw.cn/
```

如果没有配置 `auth.username` 和 `auth.password`，则不需要账号密码：

```bash
curl -x http://服务器IP:52122 http://6.ipw.cn/
curl -x socks5://服务器IP:52123 http://6.ipw.cn/
```

## 使用固定 IPv6 端口

假设配置：

```yaml
fixed:
  http_ports:
    - 52133
    - 52134
  socks5_ports:
    - 52135
```

首次启动后，程序会自动生成 `state.json`：

```json
{
  "fixed_ports": {
    "52133": "2001:399:8205:ae00:1111:2222:3333:4444",
    "52134": "2001:399:8205:ae00:5555:6666:7777:8888",
    "52135": "2001:399:8205:ae00:9999:aaaa:bbbb:cccc"
  }
}
```

之后：

- 访问 `52133` 时，始终使用 `state.json` 中 `52133` 对应的 IPv6 出口。
- 访问 `52134` 时，始终使用 `state.json` 中 `52134` 对应的 IPv6 出口。
- 访问 `52135` 时，始终使用 `state.json` 中 `52135` 对应的 IPv6 出口。

示例：

```bash
curl -x http://proxy_user:proxy_password@服务器IP:52133 http://6.ipw.cn/
curl -x http://proxy_user:proxy_password@服务器IP:52134 http://6.ipw.cn/
curl -x socks5://proxy_user:proxy_password@服务器IP:52135 http://6.ipw.cn/
```

只要不删除或修改 `state.json`，固定端口对应的 IPv6 就会保持不变。

## 认证规则

当 `auth.username` 和 `auth.password` 都为空时：

- HTTP 代理不要求认证。
- SOCKS5 代理不要求认证。

当配置了 `auth.username` 和 `auth.password` 时：

- HTTP 代理必须携带正确的 `Proxy-Authorization`。
- SOCKS5 代理必须使用用户名密码认证。
- 动态端口和固定端口都会启用同一组账号密码。

目前没有启用 IP 白名单免认证；所有非空认证配置都会对所有客户端生效。

## 固定端口数量和资源占用

固定端口数量可以配置很多，例如 200 个。空闲状态下，每个固定端口主要占用：

- 1 个监听 socket
- 1 个 goroutine
- 少量内存

真正的资源压力主要来自活跃连接数量，而不是端口数量本身。如果固定端口很多、并发也很高，建议检查系统文件描述符限制：

```bash
ulimit -n
```

必要时调高系统限制，并确保防火墙、安全组、容器端口映射已经放行这些端口。

## 注意事项

服务器必须允许使用 CIDR 内的 IPv6 作为本地出站源地址。不同系统和云厂商的网络策略不同，可能需要额外配置 IPv6 路由、IPv6 地址段或非本地地址绑定能力。

如果固定端口已经写入 `state.json`，后来修改了 `cidr`，程序会检查旧 IPv6 是否仍属于新的 CIDR；如果不属于，会自动为该端口重新分配 IPv6。

## License

MIT License，见 [LICENSE](LICENSE)。
