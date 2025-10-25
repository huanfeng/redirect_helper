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
curl "http://localhost:8001/api/update?name=test&token=<redirect_token>&target=google.com:443"

# 创建域名跳转
curl "http://localhost:8001/api/update-domain?domain=example.com&token=<domain_token>&target=https://google.com"
```

### 批量更新

支持在一次请求中更新多个跳转条目，提供 GET 和 POST 两种方式：

#### GET 方式（索引参数）

```bash
# 批量创建/更新路径跳转
curl "http://localhost:8001/api/batch-update?redirect_token=<token>&name1=test1&target1=google.com:443&name2=test2&target2=baidu.com:443"

# 批量创建/更新域名跳转
curl "http://localhost:8001/api/batch-update?domain_token=<token>&domain1=d1.example.com&target1=https://google.com&domain2=d2.example.com&target2=https://baidu.com"

# 混合批量更新（路径 + 域名）
curl "http://localhost:8001/api/batch-update?redirect_token=<r_token>&domain_token=<d_token>&name1=test&target1=google.com:443&domain2=example.com&target2=https://github.com"
```

#### POST 方式（JSON）

```bash
curl -X POST "http://localhost:8001/api/batch-update" \
  -H "Content-Type: application/json" \
  -d '{
    "redirect_token": "<redirect_token>",
    "domain_token": "<domain_token>",
    "entries": [
      {"name": "test1", "target": "google.com:443"},
      {"name": "test2", "target": "baidu.com:443"},
      {"domain": "example.com", "target": "https://github.com"}
    ]
  }'
```

**响应示例**：
```json
{
  "state": "success",
  "message": "All entries updated successfully",
  "results": [
    {"name": "test1", "target": "google.com:443", "success": true},
    {"name": "test2", "target": "baidu.com:443", "success": true}
  ],
  "summary": {
    "total": 2,
    "succeeded": 2,
    "failed": 0
  }
}
```

### 查看管理界面

访问 `http://localhost:8001` 可以使用网页界面，输入 Admin Token 查看现有的跳转配置。

## 跳转方式

- **路径跳转**: `http://localhost:8001/go/test` → `http://google.com`
- **域名跳转**: `http://example.com/any/path` → `https://google.com/any/path` (保持完整URL)


