# OneRound / 一局一分

A simple scorekeeper for family gatherings and casual card games.

## 项目简介

OneRound 是家庭聚会、朋友娱乐时使用的轻量积分记录工具。多人进入同一个牌局后，后端通过 HTTP API 保存状态，并通过 WebSocket 通知在线客户端刷新 summary。

## 技术栈

- Backend: Go 1.24, chi, database/sql, modernc.org/sqlite, goose, JWT, nhooyr websocket, slog
- Frontend: 微信原生小程序, TypeScript, WXML, WXSS
- Deploy: Nginx HTTPS/WSS, single Go server, SQLite file

## 本地启动

```bash
cd apps/server
go mod tidy
go run ./cmd/oneround-server
```

健康检查：

```bash
curl http://localhost:8080/health
```

## 数据库迁移

服务启动时会自动执行 `apps/server/migrations`。也可以只执行迁移：

```bash
cd apps/server
go run ./cmd/oneround-server -migrate-only
```

## 微信配置

本地默认启用 fake auth。生产环境创建 `apps/server/config.yaml` 或使用环境变量：

```bash
export ONEROUND_WECHAT_APP_ID="wx..."
export ONEROUND_WECHAT_APP_SECRET="..."
export ONEROUND_WECHAT_USE_FAKE_AUTH=false
export ONEROUND_AUTH_SIGNING_KEY="$(openssl rand -base64 48)"
```

真实 AppSecret 只能存在后端配置或环境变量中，不能提交到仓库。

## WebSocket 调试

先调用 `/api/auth/wechat-login` 获取 token，再连接：

```bash
wscat -c "ws://localhost:8080/ws/game-sessions/<id>?token=<jwt>"
```

提交 `/api/game-sessions/<id>/rounds` 后，同一房间客户端会收到 `round.submitted`，客户端应重新请求 summary。

## 小程序启动

用微信开发者工具打开：

```text
apps/miniprogram
```

本地调试时确认 `app.ts` 的 `baseUrl` 指向后端地址。正式环境必须使用 HTTPS/WSS 域名，并在微信小程序后台配置合法域名。

## 私有 VPC 部署

参考 `deploy/`：

- `deploy/Dockerfile`: Go 1.24 多阶段构建
- `deploy/docker-compose.yml`: 单实例服务和持久化 SQLite 数据目录
- `deploy/nginx.conf`: HTTPS 和 WSS 反向代理
- `deploy/systemd/oneround.service`: systemd 示例

## 架构限制

- SQLite 适合第一版轻量部署。
- 单实例 WebSocket 适合家庭聚会级别并发。
- 如需多实例，需要引入 Redis Pub/Sub、NATS 或其他广播通道。
- 如果写并发明显增加，需要迁移到 PostgreSQL。
- WebSocket 只是实时通知，最终状态以 HTTP summary API 为准。
- 当前版本只支持手动录入积分，不包含复杂玩法规则。
- 当前版本不处理财务结算场景。
