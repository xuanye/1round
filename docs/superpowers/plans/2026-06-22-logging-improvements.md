# Logging Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve server-side logging so expected business errors log at `info`, suspicious client/business failures log at `warn`, and request/external-call logs carry enough context for debugging.

**Architecture:** Keep business errors returned from services. Enrich the HTTP response/logging boundary with structured error metadata and request context, then add external dependency failure logging inside the WeChat adapter.

**Tech Stack:** Go 1.24, chi, zap, standard library HTTP tests

---

### Task 1: Request Log Classification

**Files:**
- Modify: `apps/server/internal/api/response/response.go`
- Modify: `apps/server/internal/api/middleware/request_log_middleware.go`
- Test: `apps/server/internal/api/middleware/request_log_middleware_test.go`

- [ ] Add failing tests for expected-business-error `info`, suspicious-auth `warn`, and `5xx` `error` logging.
- [ ] Run the middleware test package and verify the new cases fail for the current implementation.
- [ ] Add response metadata capture and log classification helpers.
- [ ] Re-run the middleware test package and verify it passes.

### Task 2: WeChat Failure Logging

**Files:**
- Modify: `apps/server/internal/infra/wechat/client.go`
- Modify: `apps/server/cmd/oneround-server/main.go`
- Test: `apps/server/internal/infra/wechat/client_test.go`

- [ ] Add failing tests that assert external failures are logged with operation context.
- [ ] Run the WeChat client test package and verify the new case fails for the current implementation.
- [ ] Inject logger dependency into the WeChat HTTP client and log upstream/network failures once.
- [ ] Re-run the WeChat client test package and verify it passes.

### Task 3: Verification

**Files:**
- None

- [ ] Run targeted tests for middleware and WeChat client.
- [ ] Run `go test ./...` for the server module.
- [ ] Review `git diff` for accidental behavior changes outside logging.
