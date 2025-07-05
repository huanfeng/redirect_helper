# Redirect Helper

Redirect Helper是一个Go语言实现的跳转辅助工具，支持两种模式：传统的URL跳转和基于域名的URL重定向。支持配置文件持久化存储，适合Docker部署。

## 功能特性

- **传统跳转模式**: 通过 `/go/name` 进行URL重定向
- **域名跳转模式**: 基于域名的URL重定向，完整保持URL路径和参数
- 命令行创建跳转名称和域名映射
- HTTP API设置跳转目标和域名目标
- 基于token的安全认证
- 配置文件持久化存储
- 命令行管理功能（创建、列出、更新、删除）
- Docker友好的配置文件路径

## 编译和安装

```bash
go build -o redirect_helper ./cmd/redirect_helper
```

## 使用方法

### 模式一：传统跳转模式

#### 1. 创建跳转名称

```bash
./redirect_helper -create abc
```

输出示例：
```
Forwarding created successfully:
Name: abc
Token: 9d3ee6a4cac9f5d3bed00cd12987fb6d
Use this token to set the target via API
Config saved to: ./redirect_helper.json
```

#### 2. 列出所有跳转

```bash
./redirect_helper -list
```

输出示例：
```
Existing forwardings:
Name: abc, Target: 192.168.1.100:8080, Created: 2025-07-05 10:28:49
```

#### 3. 通过命令行更新跳转目标

```bash
./redirect_helper -update abc -target 192.168.1.100:8080
```

#### 4. 删除跳转

```bash
./redirect_helper -remove abc
```

#### 5. 通过API设置跳转目标

```bash
curl "http://localhost:8080/api/set?name=abc&token=9d3ee6a4cac9f5d3bed00cd12987fb6d&target=1.1.1.1:12345"
```

成功响应：
```json
{"state":"success"}
```

#### 6. 使用跳转

访问以下URL将跳转到设置的目标：

```
http://localhost:8080/go/abc
```

将重定向到: `http://1.1.1.1:12345`

### 模式二：域名跳转模式（推荐）

域名跳转模式通过Host头识别请求，完整保持URL路径和查询参数进行重定向，不消耗服务器流量。

#### 1. 创建域名映射

```bash
./redirect_helper -create-domain abc.mydomain.com
```

输出示例：
```
Domain mapping created successfully:
Domain: abc.mydomain.com
Token: a01b531a0a17c66bb4c3731b64697613
Use this token to set the target via API
Config saved to: ./redirect_helper.json
```

#### 2. 列出所有域名映射

```bash
./redirect_helper -list-domains
```

输出示例：
```
Existing domain mappings:
Domain: abc.mydomain.com, Target: https://file.example.com:12345, Created: 2025-07-05 11:47:28
```

#### 3. 通过命令行设置域名目标

```bash
./redirect_helper -update-domain abc.mydomain.com -target https://file.example.com:12345
```

#### 4. 删除域名映射

```bash
./redirect_helper -remove-domain abc.mydomain.com
```

#### 5. 通过API设置域名目标

```bash
curl "http://localhost:8080/api/set-domain?domain=abc.mydomain.com&token=a01b531a0a17c66bb4c3731b64697613&target=https://file.example.com:12345"
```

成功响应：
```json
{"state":"success"}
```

#### 6. 使用域名跳转

当访问 `abc.mydomain.com/a/b/c/hello.html?param=value` 时，将重定向到 `https://file.example.com:12345/a/b/c/hello.html?param=value`

**特点**：
- ✅ 完整保持URL路径：`/a/b/c/hello.html`
- ✅ 完整保持查询参数：`?param=value`
- ✅ 支持所有HTTP方法：GET、POST、PUT、DELETE等
- ✅ 不消耗服务器流量（HTTP重定向）
- ✅ 不要求服务器高带宽
- ✅ 适合nginx前置反代

### 服务器启动

```bash
# 启动服务器（默认端口8080）
./redirect_helper -server

# 指定端口
./redirect_helper -server -port 9090

# 指定配置文件
./redirect_helper -server -config /path/to/config.json
```

## 命令行参数

### 传统跳转管理
- `-create <name>`: 创建新的跳转名称
- `-list`: 列出所有跳转条目
- `-update <name>`: 更新跳转目标（需要配合-target使用）
- `-remove <name>`: 删除跳转名称

### 域名跳转管理
- `-create-domain <domain>`: 创建新的域名映射
- `-list-domains`: 列出所有域名映射
- `-update-domain <domain>`: 更新域名目标（需要配合-target使用）
- `-remove-domain <domain>`: 删除域名映射

### 通用参数
- `-target <target>`: 指定新的目标地址
- `-server`: 启动HTTP服务器
- `-port <port>`: 指定服务器端口（默认8080）
- `-config <path>`: 指定配置文件路径（默认./redirect_helper.json）

## 配置文件

配置文件默认保存在当前目录的 `redirect_helper.json`：

```json
{
  "forwardings": {
    "abc": {
      "name": "abc",
      "token": "9d3ee6a4cac9f5d3bed00cd12987fb6d",
      "target": "192.168.1.100:8080",
      "created_at": "2025-07-05T10:28:49.028219447+08:00",
      "updated_at": "2025-07-05T10:28:56.56147355+08:00"
    }
  },
  "domains": {
    "abc.mydomain.com": {
      "domain": "abc.mydomain.com",
      "token": "a01b531a0a17c66bb4c3731b64697613",
      "target": "https://file.example.com:12345",
      "created_at": "2025-07-05T11:47:28.124467067+08:00",
      "updated_at": "2025-07-05T11:47:28.124467137+08:00"
    }
  },
  "server": {
    "port": "8080"
  }
}
```

## API 文档

### 传统跳转API

#### 设置跳转目标
- **URL**: `/api/set`
- **方法**: `GET`
- **参数**:
  - `name`: 跳转名称
  - `token`: 验证token
  - `target`: 目标地址（格式：host:port 或 完整URL）

#### 跳转
- **URL**: `/go/{name}`
- **方法**: `GET`
- **功能**: 重定向到设置的目标地址

### 域名跳转API

#### 设置域名目标
- **URL**: `/api/set-domain`
- **方法**: `GET`
- **参数**:
  - `domain`: 域名
  - `token`: 验证token
  - `target`: 目标地址（完整URL，如 https://example.com:8080）

#### 列出域名映射
- **URL**: `/api/list-domains`
- **方法**: `GET`
- **响应**: 
```json
{
  "state": "success",
  "domains": [
    {
      "domain": "abc.mydomain.com",
      "token": "a01b531a0a17c66bb4c3731b64697613",
      "target": "https://file.example.com:12345",
      "created_at": "2025-07-05T11:47:28.124467067+08:00",
      "updated_at": "2025-07-05T11:47:28.124467137+08:00"
    }
  ]
}
```

#### 域名跳转
- **URL**: `/*` (任意路径)
- **方法**: 任意HTTP方法
- **功能**: 根据Host头进行HTTP重定向

## Nginx配置示例

使用nginx作为前置反代，将不同域名转发到redirect_helper：

```nginx
# nginx.conf
upstream redirect_helper {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name *.mydomain.com;
    
    location / {
        proxy_pass http://redirect_helper;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 443 ssl;
    server_name *.mydomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://redirect_helper;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Docker部署示例

```dockerfile
# Dockerfile
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o redirect_helper ./cmd/redirect_helper

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/redirect_helper .
COPY --from=builder /app/redirect_helper.json .
EXPOSE 8080
CMD ["./redirect_helper", "-server", "-config", "./redirect_helper.json"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  redirect_helper:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./redirect_helper.json:/root/redirect_helper.json
    restart: unless-stopped
```

## 项目结构

```
redirect_helper/
├── cmd/redirect_helper/          # 主程序入口
├── internal/
│   ├── config/              # 配置文件管理
│   ├── models/              # 数据模型
│   ├── server/              # HTTP服务器
│   └── storage/             # 数据存储
├── pkg/utils/               # 工具函数
├── redirect_helper.json         # 配置文件
└── README.md
```

## 安全说明

- 每个跳转名称和域名都有唯一的token
- 只有拥有正确token的请求才能修改目标
- 目标地址会进行基本的格式验证
- 配置文件权限设置为644
- 支持HTTPS代理转发

## 数据持久化

- 使用JSON格式配置文件存储所有数据
- 配置文件位置：`./redirect_helper.json`（当前目录）
- 支持多个跳转名称和域名映射
- 自动保存创建时间和更新时间
- 支持通过 `-config` 参数自定义配置文件路径

## 使用场景对比

| 特性 | 传统跳转模式 | 域名跳转模式 |
|------|-------------|-------------|
| URL格式 | `/go/name` | 任意路径 |
| 路径保持 | ❌ 丢失 | ✅ 完整保持 |
| 参数保持 | ❌ 丢失 | ✅ 完整保持 |
| HTTP方法 | GET重定向 | ✅ 所有方法 |
| 流量消耗 | 无 | ✅ 无 |
| 带宽要求 | 低 | ✅ 低 |
| 适用场景 | 简单跳转 | 域名跳转 |
| nginx配合 | 一般 | ✅ 完美 |
| Docker部署 | 支持 | ✅ 推荐 |