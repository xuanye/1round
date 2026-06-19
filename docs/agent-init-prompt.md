# OneRound / 一局一分 项目初始化提示词

你是一个资深全栈工程 Agent。请根据本文档初始化一个微信小程序 + Go 后端项目。

项目中文名：一局一分
项目英文名：OneRound
仓库名：one-round
部署环境：用户私有 VPC
后端技术：Go 1.24 + SQLite + WebSocket
前端技术：微信原生小程序 + TypeScript
产品定位：家庭聚会、朋友娱乐时使用的多人牌局积分记录工具。
核心目标：多人在各自手机上进入同一个牌局后，可以实时看到积分变化。

---

## 1. 产品边界

这是一个休闲记分工具，不是赌博、下注、输赢结算或现金统计工具。

页面文案、代码注释、数据库字段、接口命名中禁止出现以下语义：

```text
gambling
betting
wager
casino
cash settlement
money win/loss
赌
赌局
下注
赌资
输钱
赢钱
现金结算
```

统一使用以下中性语义：

```text
score
points
round
game session
family gathering
casual card game
积分
记分
局
牌局
家庭聚会
朋友娱乐
休闲记分
```

小程序描述建议：

```text
一局一分是一款适合家庭聚会、朋友娱乐时使用的轻量记分小程序，支持多人牌局积分记录、分数统计和历史查看，让每一局计分更清楚、更方便。
```

---

## 2. 总体架构

请初始化一个 monorepo：

```text
one-round/
├─ apps/
│  ├─ miniprogram/
│  └─ server/
├─ docs/
├─ deploy/
├─ scripts/
├─ README.md
└─ .gitignore
```

系统架构：

```text
微信小程序
  ↓ HTTPS API
Go HTTP Server
  ↓
SQLite
  ↓
WebSocket Hub
  ↓
同一牌局内所有在线客户端实时收到积分变更
```

核心设计原则：

1. SQLite 是第一版主数据库。
2. 所有写操作必须通过 Go 后端完成。
3. 小程序端不能直接访问数据库。
4. 分数提交必须使用数据库事务。
5. WebSocket 只负责事件推送，不负责业务决策。
6. API 负责状态读取和写入。
7. WebSocket 断开后，小程序必须能通过 API 重新拉取最新 summary。
8. 服务必须能部署在用户私有 VPC。
9. 第一版不引入 Kubernetes。
10. 第一版不引入 Redis。
11. 第一版不引入复杂微服务。
12. 第一版不引入消息队列。
13. 第一版只支持单实例部署。
14. 后续如需多实例，需要引入 Redis Pub/Sub、NATS 或其他广播通道。

---

## 3. 技术选型

### 3.1 后端

使用 Go 1.24。

要求：

```go
module github.com/redhu/one-round/apps/server

go 1.24
```

推荐技术栈：

```text
Go: 1.24
HTTP Router: chi
Database: SQLite
SQLite Driver: modernc.org/sqlite
Database Access: database/sql
Migration: goose
WebSocket: nhooyr.io/websocket
Auth: JWT
Logger: slog
Config: YAML + environment override
Testing: Go testing + httptest
```

优先推荐组合：

```text
chi + database/sql + modernc.org/sqlite + nhooyr.io/websocket + slog + goose
```

SQLite Driver 优先使用：

```text
modernc.org/sqlite
```

原因：

1. 纯 Go 实现。
2. 避免 CGO。
3. 更适合轻量私有部署。
4. Docker 构建更简单。

第一版不要使用：

```text
Gin
GORM
Redis
PostgreSQL
Kafka
RabbitMQ
Kubernetes
复杂 Clean Architecture 模板
```

说明：

* `chi` 足够轻量。
* `database/sql` 足够可控。
* 第一版手写 SQL 可以接受，但 SQL 必须集中在 repository 层。
* Handler 中禁止散落 SQL。
* Service 中禁止直接拼接 HTTP response。
* WebSocket 推送不作为最终状态来源，最终状态以 HTTP summary API 为准。

### 3.2 前端

使用微信原生小程序：

```text
TypeScript
微信开发者工具
原生 WXML / WXSS
原生 Page / Component
```

不要使用：

```text
Taro
UniApp
React
Vue
跨端框架
复杂全局状态管理
```

小程序端职责：

1. UI 展示。
2. 登录态缓存。
3. HTTP API 调用。
4. WebSocket 连接。
5. 收到实时事件后刷新牌局 summary。
6. 本地轻量缓存最近访问牌局。

---

## 4. 后端目录结构

请在 `apps/server` 下创建：

```text
apps/server/
├─ cmd/
│  └─ oneround-server/
│     └─ main.go
├─ internal/
│  ├─ api/
│  │  ├─ handler/
│  │  │  ├─ auth_handler.go
│  │  │  ├─ game_handler.go
│  │  │  ├─ player_handler.go
│  │  │  ├─ round_handler.go
│  │  │  └─ websocket_handler.go
│  │  ├─ middleware/
│  │  │  ├─ auth_middleware.go
│  │  │  ├─ recover_middleware.go
│  │  │  └─ request_log_middleware.go
│  │  ├─ dto/
│  │  │  ├─ auth_dto.go
│  │  │  ├─ game_dto.go
│  │  │  ├─ player_dto.go
│  │  │  └─ round_dto.go
│  │  ├─ response/
│  │  │  └─ response.go
│  │  └─ router.go
│  ├─ app/
│  │  ├─ auth/
│  │  │  └─ service.go
│  │  ├─ game/
│  │  │  └─ service.go
│  │  ├─ player/
│  │  │  └─ service.go
│  │  ├─ round/
│  │  │  └─ service.go
│  │  └─ query/
│  │     └─ summary_service.go
│  ├─ domain/
│  │  ├─ user.go
│  │  ├─ game_session.go
│  │  ├─ game_member.go
│  │  ├─ player.go
│  │  ├─ round.go
│  │  └─ errors.go
│  ├─ infra/
│  │  ├─ sqlite/
│  │  │  ├─ db.go
│  │  │  ├─ user_repository.go
│  │  │  ├─ game_repository.go
│  │  │  ├─ player_repository.go
│  │  │  ├─ round_repository.go
│  │  │  └─ transaction.go
│  │  ├─ wechat/
│  │  │  ├─ client.go
│  │  │  └─ fake_client.go
│  │  ├─ auth/
│  │  │  └─ jwt_service.go
│  │  └─ clock/
│  │     └─ clock.go
│  ├─ realtime/
│  │  ├─ hub.go
│  │  ├─ client.go
│  │  ├─ event.go
│  │  └─ room.go
│  └─ config/
│     └─ config.go
├─ migrations/
├─ tests/
├─ config.example.yaml
├─ go.mod
├─ go.sum
└─ README.md
```

### 4.1 `internal/domain`

只包含领域模型、领域常量、领域错误。

禁止依赖：

```text
HTTP
SQLite
WebSocket
JWT
微信 API
```

### 4.2 `internal/app`

包含业务用例：

```text
LoginWithWechatCode
CreateGameSession
JoinGameSession
AddPlayer
UpdatePlayer
RemovePlayer
SubmitRoundScore
GetGameSummary
GetGameRanking
GetRecentRounds
FinishGameSession
```

### 4.3 `internal/infra`

包含基础设施实现：

```text
SQLite 连接
Repository 实现
微信 API Client
JWT Token Service
Clock Service
配置读取
```

### 4.4 `internal/api`

包含 HTTP 层：

```text
HTTP handler
DTO
middleware
response envelope
router
```

### 4.5 `internal/realtime`

包含 WebSocket 层：

```text
WebSocket Hub
牌局房间管理
客户端连接管理
广播事件
心跳
断线清理
```

---

## 5. Domain Model

### 5.1 User

```go
package domain

import "time"

type User struct {
	ID          string
	OpenID      string
	UnionID      *string
	DisplayName *string
	AvatarURL    *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

### 5.2 GameSession

```go
package domain

import "time"

type GameSessionStatus string

const (
	GameSessionStatusActive   GameSessionStatus = "active"
	GameSessionStatusFinished GameSessionStatus = "finished"
)

type GameSession struct {
	ID              string
	Name            string
	InviteCode      string
	OwnerUserID     string
	Status          GameSessionStatus
	ZeroSumRequired bool
	RoundCount      int
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

### 5.3 GameMember

```go
package domain

import "time"

type GameMemberRole string

const (
	GameMemberRoleOwner  GameMemberRole = "owner"
	GameMemberRoleMember GameMemberRole = "member"
)

type GameMember struct {
	ID            string
	GameSessionID string
	UserID        string
	Role          GameMemberRole
	JoinedAt      time.Time
}
```

### 5.4 Player

```go
package domain

import "time"

type Player struct {
	ID            string
	GameSessionID string
	UserID        *string
	DisplayName   string
	TotalScore    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
```

### 5.5 Round

```go
package domain

import "time"

type Round struct {
	ID              string
	GameSessionID   string
	RoundNo         int
	CreatedByUserID string
	Note            *string
	CreatedAt       time.Time
	Scores          []RoundScore
}

type RoundScore struct {
	ID       string
	RoundID  string
	PlayerID string
	Score    int
}
```

### 5.6 Domain Errors

```go
package domain

import "errors"

var (
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrNotFound              = errors.New("not found")
	ErrGameSessionFinished   = errors.New("game session finished")
	ErrScoreTotalMustBeZero  = errors.New("score total must be zero")
	ErrInvalidPlayer         = errors.New("invalid player")
	ErrPlayerAlreadyExists   = errors.New("player already exists")
	ErrGameMemberRequired    = errors.New("game member required")
)
```

---

## 6. SQLite Schema

请创建 goose migrations。

目录：

```text
apps/server/migrations/
├─ 00001_create_users.sql
├─ 00002_create_game_sessions.sql
├─ 00003_create_players.sql
├─ 00004_create_rounds.sql
└─ 00005_create_game_members.sql
```

### 6.1 users

```sql
-- +goose Up
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    open_id TEXT NOT NULL UNIQUE,
    union_id TEXT NULL,
    display_name TEXT NULL,
    avatar_url TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_users_open_id ON users(open_id);

-- +goose Down
DROP TABLE users;
```

### 6.2 game_sessions

```sql
-- +goose Up
CREATE TABLE game_sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    invite_code TEXT NOT NULL UNIQUE,
    owner_user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    zero_sum_required INTEGER NOT NULL,
    round_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(owner_user_id) REFERENCES users(id)
);

CREATE INDEX idx_game_sessions_owner_user_id ON game_sessions(owner_user_id);
CREATE INDEX idx_game_sessions_invite_code ON game_sessions(invite_code);
CREATE INDEX idx_game_sessions_updated_at ON game_sessions(updated_at);

-- +goose Down
DROP TABLE game_sessions;
```

### 6.3 players

```sql
-- +goose Up
CREATE TABLE players (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    user_id TEXT NULL,
    display_name TEXT NOT NULL,
    total_score INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX idx_players_game_session_id ON players(game_session_id);
CREATE INDEX idx_players_user_id ON players(user_id);

-- +goose Down
DROP TABLE players;
```

### 6.4 rounds / round_scores

```sql
-- +goose Up
CREATE TABLE rounds (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    round_no INTEGER NOT NULL,
    created_by_user_id TEXT NOT NULL,
    note TEXT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(created_by_user_id) REFERENCES users(id),
    UNIQUE(game_session_id, round_no)
);

CREATE INDEX idx_rounds_game_session_id_round_no 
ON rounds(game_session_id, round_no DESC);

CREATE TABLE round_scores (
    id TEXT PRIMARY KEY,
    round_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    score INTEGER NOT NULL,
    FOREIGN KEY(round_id) REFERENCES rounds(id),
    FOREIGN KEY(player_id) REFERENCES players(id)
);

CREATE INDEX idx_round_scores_round_id ON round_scores(round_id);
CREATE INDEX idx_round_scores_player_id ON round_scores(player_id);

-- +goose Down
DROP TABLE round_scores;
DROP TABLE rounds;
```

### 6.5 game_members

```sql
-- +goose Up
CREATE TABLE game_members (
    id TEXT PRIMARY KEY,
    game_session_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    joined_at TEXT NOT NULL,
    FOREIGN KEY(game_session_id) REFERENCES game_sessions(id),
    FOREIGN KEY(user_id) REFERENCES users(id),
    UNIQUE(game_session_id, user_id)
);

CREATE INDEX idx_game_members_game_session_id ON game_members(game_session_id);
CREATE INDEX idx_game_members_user_id ON game_members(user_id);

-- +goose Down
DROP TABLE game_members;
```

---

## 7. 核心业务规则

必须实现以下规则：

1. 一个牌局由一个用户创建。
2. 创建牌局时自动生成 `InviteCode`。
3. `InviteCode` 使用大写字母和数字，长度 6。
4. 创建者自动成为 `game_members.owner`。
5. 用户可通过 `InviteCode` 加入牌局。
6. 一个牌局可以有多个玩家。
7. 玩家不一定绑定微信用户，允许手动添加家庭成员名称。
8. 提交一局分数时，必须给当前牌局内所有玩家提交分数，除非未来明确支持缺席。
9. 如果 `ZeroSumRequired = true`，则一局所有玩家分数总和必须为 0。
10. 每次提交分数时必须在数据库事务中完成：

    * 新增 `rounds`
    * 新增 `round_scores`
    * 累加 `players.total_score`
    * `game_sessions.round_count + 1`
    * `game_sessions.version + 1`
    * 更新 `game_sessions.updated_at`
11. 已结束的牌局不能再提交分数。
12. 排行榜按照 `total_score DESC` 排序。
13. 用户只有加入牌局后才能读取该牌局详情。
14. 只有牌局成员才能通过 WebSocket 订阅该牌局。
15. WebSocket 推送事件只作为通知，客户端收到后应重新拉取 summary 或局部刷新。
16. 删除玩家只允许在玩家没有任何 `round_scores` 时进行。
17. 结束牌局后仍允许读取 summary、ranking、recent rounds。
18. 结束牌局后禁止添加玩家、修改玩家、提交分数。

---

## 8. API Response

统一响应结构：

```go
package response

type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *T     `json:"data"`
}
```

成功：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

错误：

```json
{
  "code": 40001,
  "message": "invalid argument",
  "data": null
}
```

错误码建议：

```text
0      ok
40001  invalid argument
40002  validation failed
40101  unauthorized
40301  forbidden
40401  not found
40901  conflict
50001  internal error
```

---

## 9. HTTP API 设计

### 9.1 API 列表

```text
POST /api/auth/wechat-login

POST /api/game-sessions
GET  /api/game-sessions/{id}
GET  /api/game-sessions/{id}/summary
POST /api/game-sessions/join
POST /api/game-sessions/{id}/finish

POST   /api/game-sessions/{id}/players
GET    /api/game-sessions/{id}/players
PATCH  /api/game-sessions/{id}/players/{playerId}
DELETE /api/game-sessions/{id}/players/{playerId}

POST /api/game-sessions/{id}/rounds
GET  /api/game-sessions/{id}/rounds/recent
GET  /api/game-sessions/{id}/ranking

GET /health
GET /ws/game-sessions/{id}
```

---

## 10. 登录设计

### 10.1 POST `/api/auth/wechat-login`

Request:

```json
{
  "code": "wx-login-code"
}
```

Response:

```json
{
  "token": "jwt-token",
  "user": {
    "id": "uuid",
    "displayName": null,
    "avatarUrl": null
  }
}
```

要求：

1. 小程序端调用 `wx.login()` 获取 code。
2. 后端调用微信 `jscode2session` 换 openid。
3. AppSecret 只能存在后端配置中。
4. 开发环境支持 fake auth。
5. fake auth 模式下，允许传入任意 code 并映射为稳定 openid。
6. 登录成功后，后端签发 JWT。
7. 小程序后续 HTTP 请求使用：

```http
Authorization: Bearer <token>
```

8. WebSocket 也必须携带 token。

---

## 11. Game Session API

### 11.1 POST `/api/game-sessions`

Request:

```json
{
  "name": "家庭聚会",
  "zeroSumRequired": true
}
```

Response:

```json
{
  "id": "uuid",
  "name": "家庭聚会",
  "inviteCode": "A8K21P",
  "status": "active",
  "zeroSumRequired": true,
  "roundCount": 0,
  "version": 1
}
```

### 11.2 POST `/api/game-sessions/join`

Request:

```json
{
  "inviteCode": "A8K21P"
}
```

Response:

```json
{
  "gameSessionId": "uuid"
}
```

### 11.3 GET `/api/game-sessions/{id}`

Response:

```json
{
  "id": "uuid",
  "name": "家庭聚会",
  "inviteCode": "A8K21P",
  "status": "active",
  "zeroSumRequired": true,
  "roundCount": 12,
  "version": 15,
  "createdAt": "2026-06-19T00:00:00Z",
  "updatedAt": "2026-06-19T00:00:00Z"
}
```

### 11.4 POST `/api/game-sessions/{id}/finish`

Response:

```json
{
  "id": "uuid",
  "status": "finished"
}
```

---

## 12. Player API

### 12.1 POST `/api/game-sessions/{id}/players`

Request:

```json
{
  "displayName": "爸爸"
}
```

Response:

```json
{
  "id": "uuid",
  "displayName": "爸爸",
  "totalScore": 0
}
```

### 12.2 GET `/api/game-sessions/{id}/players`

Response:

```json
[
  {
    "id": "uuid",
    "displayName": "爸爸",
    "totalScore": 36
  }
]
```

### 12.3 PATCH `/api/game-sessions/{id}/players/{playerId}`

Request:

```json
{
  "displayName": "爸爸"
}
```

Response:

```json
{
  "id": "uuid",
  "displayName": "爸爸",
  "totalScore": 36
}
```

### 12.4 DELETE `/api/game-sessions/{id}/players/{playerId}`

规则：

1. 如果玩家已经参与过计分，则不能删除。
2. 如果玩家未参与任何 round score，则允许删除。

Response:

```json
{
  "deleted": true
}
```

---

## 13. Round API

### 13.1 POST `/api/game-sessions/{id}/rounds`

Request:

```json
{
  "scores": [
    {
      "playerId": "uuid",
      "score": 10
    },
    {
      "playerId": "uuid",
      "score": -10
    }
  ],
  "note": "第1局"
}
```

Response:

```json
{
  "roundId": "uuid",
  "roundNo": 1,
  "version": 2
}
```

处理流程：

```text
校验登录
校验用户是牌局成员
校验牌局 active
读取当前玩家列表
校验 scores 覆盖所有玩家
校验所有 playerId 属于当前牌局
如果 zeroSumRequired=true，校验总分为 0
开启事务
新增 round
新增 round_scores
累加 players.total_score
更新 game_sessions.round_count
更新 game_sessions.version
更新 game_sessions.updated_at
提交事务
广播 round.submitted WebSocket 事件
返回 roundNo/version
```

### 13.2 GET `/api/game-sessions/{id}/rounds/recent`

Query:

```text
limit=20
```

Response:

```json
[
  {
    "id": "uuid",
    "roundNo": 12,
    "createdAt": "2026-06-19T00:00:00Z",
    "scores": [
      {
        "playerId": "uuid",
        "score": 10
      }
    ]
  }
]
```

---

## 14. Summary / Ranking API

### 14.1 GET `/api/game-sessions/{id}/summary`

Response:

```json
{
  "id": "uuid",
  "name": "家庭聚会",
  "status": "active",
  "roundCount": 12,
  "zeroSumRequired": true,
  "players": [
    {
      "id": "uuid",
      "displayName": "爸爸",
      "totalScore": 36,
      "averageScore": 3
    }
  ],
  "recentRounds": [
    {
      "id": "uuid",
      "roundNo": 12,
      "createdAt": "2026-06-19T00:00:00Z",
      "scores": [
        {
          "playerId": "uuid",
          "score": 10
        }
      ]
    }
  ],
  "updatedAt": "2026-06-19T00:00:00Z",
  "version": 15
}
```

`version` 使用 `game_sessions.version`。

### 14.2 GET `/api/game-sessions/{id}/ranking`

Response:

```json
[
  {
    "rank": 1,
    "playerId": "uuid",
    "displayName": "爸爸",
    "totalScore": 36,
    "roundCount": 12,
    "averageScore": 3
  }
]
```

排序规则：

```text
totalScore DESC
displayName ASC
```

---

## 15. WebSocket 设计

### 15.1 连接地址

```text
GET /ws/game-sessions/{id}
```

Header：

```http
Authorization: Bearer <token>
```

如果小程序 WebSocket API 不方便设置 Header，则支持 query token：

```text
wss://example.com/ws/game-sessions/{id}?token=<jwt>
```

优先使用 Header；query token 仅作为小程序兼容方案。

### 15.2 连接规则

1. 必须登录。
2. 必须是该牌局成员。
3. 一个用户可以有多个连接。
4. 服务端按 `gameSessionId` 管理 room。
5. 客户端进入牌局详情页时连接 WebSocket。
6. 客户端离开牌局详情页时关闭 WebSocket。
7. 客户端断线后要自动重连。
8. 重连成功后必须重新拉取 summary。

### 15.3 事件结构

```go
package realtime

import "time"

type Event struct {
	Type          string      `json:"type"`
	GameSessionID string      `json:"gameSessionId"`
	Version       int64       `json:"version"`
	Payload       any         `json:"payload,omitempty"`
	SentAt        time.Time   `json:"sentAt"`
}
```

### 15.4 事件类型

```text
game.updated
player.added
player.updated
player.removed
round.submitted
game.finished
```

### 15.5 `round.submitted` 示例

```json
{
  "type": "round.submitted",
  "gameSessionId": "uuid",
  "version": 15,
  "payload": {
    "roundNo": 12
  },
  "sentAt": "2026-06-19T00:00:00Z"
}
```

### 15.6 客户端处理策略

收到以下事件后，客户端重新拉取 summary：

```text
game.updated
player.added
player.updated
player.removed
round.submitted
game.finished
```

第一版不要在 WebSocket payload 中传完整排行榜，避免状态不一致。

---

## 16. WebSocket Hub 实现要求

请实现：

```go
type Hub interface {
	Register(ctx context.Context, gameSessionID string, client *Client) error
	Unregister(gameSessionID string, client *Client)
	BroadcastToGame(ctx context.Context, gameSessionID string, event Event)
	Close(ctx context.Context) error
}
```

必须支持：

```text
Register
Unregister
BroadcastToGame
Close
心跳 ping/pong
客户端读循环
客户端写循环
写超时
上下文取消
服务关闭时优雅断开
```

并发要求：

1. Hub 内部必须使用 mutex 或 channel 模型保证并发安全。
2. 广播不能因为一个慢客户端阻塞整个 room。
3. 慢客户端可断开。
4. 客户端写队列需要有容量限制。
5. 服务关闭时要关闭所有连接。

推荐 Client 结构：

```go
type Client struct {
	ID            string
	UserID        string
	GameSessionID string
	Conn          *websocket.Conn
	Send          chan Event
}
```

---

## 17. 配置文件

请创建：

```text
apps/server/config.example.yaml
```

内容：

```yaml
server:
  env: development
  http_addr: ":8080"
  public_base_url: "http://localhost:8080"

database:
  path: "./data/oneround.db"

wechat:
  app_id: ""
  app_secret: ""
  use_fake_auth: true

auth:
  signing_key: "dev-only-change-me"
  token_ttl_hours: 720

realtime:
  write_timeout_seconds: 10
  pong_timeout_seconds: 60
  ping_interval_seconds: 30
  client_send_queue_size: 32
```

要求：

1. 真实 AppSecret 不允许提交到 git。
2. 支持环境变量覆盖配置。
3. README 必须说明如何配置微信 AppId/AppSecret。
4. 本地开发默认启用 fake auth。
5. 生产环境必须关闭 fake auth。
6. 生产环境必须使用强随机 JWT signing key。

---

## 18. 微信小程序结构

请在 `apps/miniprogram` 下初始化微信原生小程序：

```text
apps/miniprogram/
├─ app.ts
├─ app.json
├─ app.wxss
├─ sitemap.json
├─ pages/
│  ├─ home/
│  │  ├─ index.ts
│  │  ├─ index.wxml
│  │  ├─ index.wxss
│  │  └─ index.json
│  ├─ game-create/
│  ├─ game-join/
│  ├─ game-detail/
│  ├─ player-manage/
│  ├─ score-input/
│  ├─ ranking/
│  └─ history/
├─ components/
│  ├─ score-card/
│  ├─ player-list/
│  ├─ round-list/
│  └─ empty-state/
├─ services/
│  ├─ http.ts
│  ├─ auth.service.ts
│  ├─ game.service.ts
│  ├─ score.service.ts
│  └─ realtime.service.ts
├─ models/
│  ├─ user.ts
│  ├─ game-session.ts
│  ├─ player.ts
│  ├─ round.ts
│  └─ realtime-event.ts
└─ utils/
   ├─ storage.ts
   └─ format.ts
```

---

## 19. 小程序页面职责

### 19.1 home

显示：

```text
一局一分
家庭聚会轻松记分
```

按钮：

```text
开始记分
加入牌局
历史记录
```

### 19.2 game-create

功能：

```text
输入牌局名称
选择是否要求总分为 0
创建成功后进入 game-detail
```

默认值：

```text
name: 家庭聚会
zeroSumRequired: true
```

### 19.3 game-join

功能：

```text
输入 inviteCode
加入牌局
进入 game-detail
```

### 19.4 game-detail

功能：

```text
显示玩家总分排行
显示最近几局
添加玩家
记录一局
分享加入
结束牌局
```

生命周期：

```text
onLoad:
  拉取 summary
  连接 WebSocket

onShow:
  如果没有连接，重连 WebSocket
  拉取 summary

收到 WebSocket 事件:
  拉取 summary

onHide:
  可保持连接或关闭连接，第一版建议关闭

onUnload:
  关闭 WebSocket
```

### 19.5 player-manage

功能：

```text
添加玩家
删除未参与计分的玩家
修改玩家显示名
```

### 19.6 score-input

功能：

```text
显示所有玩家
输入每个玩家本局分数
如果 zeroSumRequired=true，实时显示当前总和
总和不为 0 时禁止提交
提交成功后返回 game-detail
```

### 19.7 ranking

功能：

```text
显示排名
玩家名
总分
局均分
```

### 19.8 history

功能：

```text
显示用户创建或加入过的历史牌局
```

---

## 20. 小程序 Service 设计

### 20.1 HTTP Service

```ts
export type ApiResponse<T> = {
  code: number;
  message: string;
  data: T | null;
};

export async function request<T>(options: {
  url: string;
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  data?: unknown;
  auth?: boolean;
}): Promise<T> {
  // 统一 baseUrl
  // 统一 Authorization
  // 统一错误处理
}
```

### 20.2 Auth Service

```ts
export async function login(): Promise<void> {
  // wx.login()
  // POST /api/auth/wechat-login
  // 保存 token
}
```

### 20.3 Realtime Service

```ts
export type RealtimeEvent = {
  type: string;
  gameSessionId: string;
  version: number;
  payload?: unknown;
  sentAt: string;
};

export class RealtimeService {
  connect(gameSessionId: string): void;
  disconnect(): void;
  onEvent(handler: (event: RealtimeEvent) => void): void;
}
```

要求：

1. 页面不要直接调用 `wx.connectSocket`。
2. 必须通过 `RealtimeService` 封装。
3. 支持自动重连。
4. 重连后触发页面重新拉取 summary。
5. 页面卸载必须关闭连接。
6. token 过期时跳转重新登录。

---

## 21. 小程序 UI 风格

推荐色彩：

```text
primary: #16A34A
accent: #FACC15
background: #F8FAFC
surface: #FFFFFF
text: #0F172A
muted: #64748B
danger: #EF4444
border: #E2E8F0
```

设计要求：

1. 大按钮。
2. 大数字。
3. 高对比度。
4. 少输入。
5. 少配置。
6. 不使用复杂动画。
7. 不使用赌博筹码、赌场、现金图标。
8. 可以使用纸牌、积分、计分板、家庭聚会相关中性图形。

---

## 22. 私有 VPC 部署要求

请在 `deploy/` 下提供：

```text
deploy/
├─ Dockerfile
├─ docker-compose.yml
├─ nginx.conf
├─ systemd/
│  └─ oneround.service
└─ README.md
```

部署目标：

```text
Nginx
  ↓ HTTPS / WSS
OneRound Go Server
  ↓
SQLite database file
```

要求：

1. Go 服务监听内网端口，例如 `127.0.0.1:8080`。
2. Nginx 对外提供 HTTPS。
3. WebSocket 使用 WSS。
4. SQLite 数据文件挂载到持久化目录。
5. 提供 systemd 部署方式。
6. 提供 Docker Compose 部署方式。
7. 不要把 SQLite 数据库文件打进镜像。
8. 需要有 `/health` 健康检查。
9. Nginx 必须包含 WebSocket upgrade 配置。
10. 小程序正式环境必须使用 HTTPS/WSS 域名。
11. 域名需要配置到微信小程序后台合法域名中。

### 22.1 Dockerfile

必须使用 Go 1.24：

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY apps/server/go.mod apps/server/go.sum ./
RUN go mod download

COPY apps/server/ ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/oneround-server ./cmd/oneround-server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /out/oneround-server /app/oneround-server
COPY apps/server/config.example.yaml /app/config.example.yaml
COPY apps/server/migrations /app/migrations

RUN mkdir -p /app/data

EXPOSE 8080

ENTRYPOINT ["/app/oneround-server"]
```

### 22.2 Nginx WebSocket 配置

```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /etc/nginx/certs/fullchain.pem;
    ssl_certificate_key /etc/nginx/certs/privkey.pem;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://127.0.0.1:8080;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 3600s;
    }
}
```

---

## 23. 测试要求

### 23.1 Unit Tests

至少创建以下测试：

```text
InviteCode 生成长度正确且字符合法
zero-sum 校验
已结束牌局禁止提交分数
非成员不能读取牌局
提交分数后玩家总分正确累加
排行榜排序正确
WebSocket Hub register/unregister 正确
BroadcastToGame 只广播到指定牌局
```

### 23.2 Integration Tests

至少创建以下测试：

```text
fake auth login
create game session
join game session
add players
submit round score
get summary
get ranking
WebSocket 连接后提交分数能收到 round.submitted 事件
```

### 23.3 验证命令

```bash
cd apps/server
go test ./...
go run ./cmd/oneround-server
```

---

## 24. README 要求

根目录 README 必须包含：

```text
# OneRound / 一局一分

A simple scorekeeper for family gatherings and casual card games.
```

并包含：

1. 项目简介。
2. 技术栈。
3. 本地启动方式。
4. 小程序启动方式。
5. 后端启动方式。
6. 数据库迁移方式。
7. 微信 AppId/AppSecret 配置方式。
8. WebSocket 调试方式。
9. 私有 VPC 部署方式。
10. Nginx HTTPS/WSS 配置说明。
11. 审核合规注意事项。
12. 已知架构限制。

---

## 25. 初始化命令建议

请生成可执行初始化脚本或说明。

### 25.1 创建目录

```bash
mkdir -p one-round
cd one-round

mkdir -p apps/miniprogram
mkdir -p apps/server
mkdir -p docs
mkdir -p deploy/systemd
mkdir -p scripts
```

### 25.2 初始化 Go 后端

```bash
cd apps/server

go mod init github.com/redhu/one-round/apps/server

mkdir -p cmd/oneround-server
mkdir -p internal/api/{handler,middleware,dto,response}
mkdir -p internal/app/{auth,game,player,round,query}
mkdir -p internal/domain
mkdir -p internal/infra/{sqlite,wechat,auth,clock}
mkdir -p internal/realtime
mkdir -p internal/config
mkdir -p migrations
mkdir -p tests
```

### 25.3 添加依赖

```bash
go get github.com/go-chi/chi/v5
go get github.com/google/uuid
go get github.com/golang-jwt/jwt/v5
go get modernc.org/sqlite
go get nhooyr.io/websocket
go get github.com/pressly/goose/v3
go get gopkg.in/yaml.v3
```

### 25.4 go.mod 要求

`apps/server/go.mod` 必须为：

```go
module github.com/redhu/one-round/apps/server

go 1.24
```

---

## 26. 第一阶段验收标准

完成初始化后必须满足：

1. `go.mod` 使用 `go 1.24`。
2. Dockerfile 使用 `golang:1.24-alpine`。
3. `go test ./...` 通过。
4. `go run ./cmd/oneround-server` 可启动。
5. `/health` 返回正常。
6. SQLite 数据库可通过 migration 创建。
7. fake auth 模式下可以完成：

   * 登录
   * 创建牌局
   * 加入牌局
   * 添加玩家
   * 提交一局分数
   * 查看 summary
   * 查看 ranking
8. WebSocket 能连接。
9. 提交分数后，同一个 `gameSessionId` 房间内客户端收到 `round.submitted` 事件。
10. 小程序项目可在微信开发者工具中打开。
11. 小程序页面至少包含：

    * 首页
    * 创建牌局页
    * 加入牌局页
    * 牌局详情页
    * 录分页
12. 小程序服务层已封装：

    * HTTP 请求
    * Auth
    * Game API
    * Score API
    * Realtime WebSocket
13. README 有完整启动说明。
14. deploy 目录有 Docker、Nginx、systemd 示例。
15. 文案没有赌博、下注、现金结算等敏感语义。

---

## 27. 实现优先级

请按以下顺序实现：

1. 创建 monorepo 目录结构。
2. 初始化 Go module。
3. 固定 Go 版本为 1.24。
4. 创建配置加载。
5. 创建 SQLite 连接。
6. 创建 migrations。
7. 定义 Domain models。
8. 实现 fake auth login。
9. 实现 JWT token service。
10. 实现 API response envelope。
11. 实现 create game session。
12. 实现 join game session。
13. 实现 add player。
14. 实现 submit round score transaction。
15. 实现 get summary / ranking。
16. 实现 WebSocket Hub。
17. 实现 WebSocket auth。
18. 提交分数后广播 `round.submitted`。
19. 初始化微信小程序目录。
20. 实现小程序 http service。
21. 实现小程序 auth service。
22. 实现小程序 realtime service。
23. 实现首页。
24. 实现创建牌局页。
25. 实现加入牌局页。
26. 实现牌局详情页。
27. 实现录分页。
28. 补充测试。
29. 补充 README。
30. 补充 deploy 目录。

---

## 28. 代码风格

### 28.1 后端

要求：

1. 使用 `context.Context`。
2. Handler 不写业务逻辑。
3. Service 不依赖 HTTP。
4. Repository 不依赖 Service。
5. SQL 不散落在 Handler 中。
6. 统一错误码。
7. 统一响应结构。
8. 统一日志。
9. 明确命名，不使用无意义缩写。
10. 不使用全局可变状态，WebSocket Hub 除外。
11. 对 SQLite 写事务加明确边界。
12. WebSocket 客户端必须有关闭路径。
13. 所有外部输入必须校验。
14. 所有时间统一使用 UTC。
15. JSON 字段使用 lower camelCase。
16. 数据库字段使用 snake_case。

### 28.2 小程序

要求：

1. TypeScript。
2. Service 层统一请求。
3. 页面只处理交互和展示。
4. 不在页面里散落 API URL。
5. token 统一存储。
6. WebSocket 统一封装。
7. 页面卸载必须关闭 WebSocket。
8. 断线重连后重新拉取 summary。
9. 不做复杂全局状态管理。
10. 所有用户输入做基本校验。
11. 分数输入必须转换为整数。
12. zero-sum 模式下，总和不为 0 禁止提交。

---

## 29. 暂不实现

第一版不要实现：

```text
微信支付
金钱结算
现金统计
复杂玩法规则引擎
Redis
Kafka
RabbitMQ
Kubernetes
多实例 WebSocket 横向扩展
好友系统
群聊系统
图片上传
头像上传
积分兑换
排行榜分享海报
复杂后台管理系统
后台管理端
运营统计
```

---

## 30. 已知架构限制

请在 README 中明确说明：

1. SQLite 适合第一版轻量部署。
2. 单实例 WebSocket 适合家庭聚会级别并发。
3. 如果未来需要多实例部署，需要引入 Redis Pub/Sub、NATS 或其他广播通道。
4. 如果未来写并发明显增加，需要迁移到 PostgreSQL。
5. WebSocket 推送只是实时通知，最终状态以 HTTP summary API 返回为准。
6. 当前版本不支持复杂玩法规则，只支持手动录入积分。
7. 当前版本不处理金钱、下注、输赢结算等场景。

---

## 31. 输出要求

完成项目初始化后，请输出：

1. 创建的目录结构。
2. 关键命令。
3. 关键文件说明。
4. 本地启动步骤。
5. WebSocket 测试方式。
6. SQLite migration 执行方式。
7. 私有 VPC 部署说明。
8. 小程序开发者工具打开方式。
9. 后续待办清单。
10. 已知限制。

不要只生成说明文档，必须实际创建项目骨架和关键代码。

---

## 32. 最小可运行闭环

第一阶段必须跑通以下闭环：

```text
用户打开小程序
  ↓
wx.login 获取 code
  ↓
后端 fake auth / wechat auth 登录
  ↓
小程序保存 token
  ↓
用户创建牌局
  ↓
用户添加 4 个玩家
  ↓
其他用户通过 inviteCode 加入牌局
  ↓
所有用户进入牌局详情页并连接 WebSocket
  ↓
一个用户提交第 1 局分数
  ↓
后端事务写入 SQLite
  ↓
后端广播 round.submitted
  ↓
所有在线用户收到事件
  ↓
小程序重新拉取 summary
  ↓
所有手机显示最新积分
```

