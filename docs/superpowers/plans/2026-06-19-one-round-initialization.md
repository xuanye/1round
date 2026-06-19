# OneRound Initialization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Initialize a runnable WeChat Mini Program plus Go backend monorepo for OneRound.

**Architecture:** Single-instance Go HTTP server owns all writes to SQLite and emits WebSocket notifications per game session room. The Mini Program uses service wrappers for HTTP, auth, score submission, and realtime notifications.

**Tech Stack:** Go 1.24, chi, database/sql, modernc.org/sqlite, nhooyr websocket, JWT, goose migrations, WeChat Mini Program TypeScript.

---

### Task 1: Repository Skeleton

**Files:**
- Create root docs, deploy, scripts, app folders, README, `.gitignore`.

- [x] **Step 1: Create monorepo directories**
Run: `mkdir -p apps/server apps/miniprogram docs deploy/systemd scripts`

- [x] **Step 2: Add repository docs**
Create README and deployment docs with local, migration, WebSocket, Mini Program, and VPC instructions.

### Task 2: Backend Minimal Closed Loop

**Files:**
- Create: `apps/server/go.mod`
- Create: `apps/server/cmd/oneround-server/main.go`
- Create: `apps/server/internal/**`
- Create: `apps/server/migrations/*.sql`
- Create: `apps/server/tests/*_test.go`

- [x] **Step 1: Add failing behavior tests**
Cover invite code format, zero-sum validation, finished session protection, membership reads, score accumulation, ranking, and WebSocket hub room isolation.

- [x] **Step 2: Implement domain, repositories, services, handlers**
Keep SQL in repository layer and score writes inside one transaction.

- [x] **Step 3: Verify backend**
Run: `cd apps/server && go test ./...`

### Task 3: Mini Program Skeleton

**Files:**
- Create: `apps/miniprogram/app.ts`
- Create: `apps/miniprogram/pages/**`
- Create: `apps/miniprogram/services/**`
- Create: `apps/miniprogram/models/**`
- Create: `apps/miniprogram/components/**`

- [x] **Step 1: Add required pages**
Create home, create, join, detail, player management, score input, ranking, and history pages.

- [x] **Step 2: Add service wrappers**
Encapsulate HTTP, auth, game, score, and WebSocket logic outside pages.

### Task 4: Deployment Assets

**Files:**
- Create: `deploy/Dockerfile`
- Create: `deploy/docker-compose.yml`
- Create: `deploy/nginx.conf`
- Create: `deploy/systemd/oneround.service`
- Create: `deploy/README.md`

- [x] **Step 1: Add private VPC deployment examples**
Support Docker Compose and systemd, with Nginx HTTPS/WSS proxy configuration.
