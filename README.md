# Redirect Helper

Redirect Helper是一个Go语言实现的跳转辅助工具，支持两种模式：传统的URL跳转和基于域名的URL重定向。

## 快速开始

### Docker 启动

```bash
# 或直接运行
docker run -p 8001:8001 -v ./config:/app/config redirect_helper
```

### Docker Compose 启动

```yaml
# docker-compose.yml
services:
  redirect_helper:
    image: huanfengf/redirect_helper
    contain_name: redirect_helper
    ports:
      - "8001:8001"
    volumes:
      - ./config/:/app/config/
    restart: unless-stopped
```
```bash
# 使用 docker-compose 启动
docker-compose up -d
```

### 首次运行

第一次运行时，会自动生成配置文件和tokens。**请查看日志获取tokens**：

```bash
# 查看日志获取tokens
docker-compose logs redirect_helper

# 或
docker logs <container_id>
```

日志中会显示生成的tokens，类似：
```
Admin Token: ae51c4469adcd481ec8a0962ea3dd86a
Redirect Token: 82e164cbc1d0bdf1feacf7153ff80fd4
Domain Token: 9e4c42fe94b24058037f2cd8a042a267
```

### Token 作用说明

- **Admin Token**: 管理操作 (查看列表、删除条目)
- **Redirect Token**: 创建/更新路径跳转 (`/go/name`)
- **Domain Token**: 创建/更新域名跳转

## API 使用

### 创建/更新跳转

```bash
# 创建路径跳转
curl "http://localhost:8001/api/update?name=test&token=<redirect_token>&target=google.com"

# 创建域名跳转
curl "http://localhost:8001/api/update-domain?domain=example.com&token=<domain_token>&target=https://google.com"
```

### 查看管理界面

访问 `http://localhost:8001` 可以使用网页界面，输入 Admin Token 查看现有的跳转配置。

## 跳转方式

- **路径跳转**: `http://localhost:8001/go/test` → `http://google.com`
- **域名跳转**: `http://example.com/any/path` → `https://google.com/any/path` (保持完整URL)


