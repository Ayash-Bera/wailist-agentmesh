# What's Left

Phase 1 backend is complete and deployed. This document tracks remaining gaps before the product is fully functional.

---

## High priority (blocks real usage)

### 1. Wire LogDrawer to SSE stream

**File:** `frontend/src/components/canvas/LogDrawer.tsx`

LogDrawer currently renders hardcoded mock log lines from `data.ts`. It needs to:

1. Accept a `runId` prop from `CanvasPage`
2. Open an `EventSource` to `GET /runs/{runId}/stream` when a run starts
3. Append incoming `event: log` payloads to a local state array
4. Close the stream on `event: done`

`api.ts` also needs a `runs.stream(runId)` helper that returns an `EventSource`.

### 2. Real JWT auth

**Files:** `backend/internal/api/handlers/auth.go`, `backend/internal/api/middleware.go`

Phase 1 auth stubs:
- `POST /auth/signin` and `/auth/signup` always return `"dev-token"` regardless of credentials
- Middleware accepts any Bearer token and maps every request to `userID = "dev"`

Phase 2 needs:
- Password hashing (`bcrypt`) on signup, verification on signin
- Real JWT signing and validation (e.g. `golang-jwt/jwt`)
- Per-user data isolation (currently all data is owned by `"dev"`)
- The `users` table already exists in the migration — just needs the handlers wired

### 3. StopWorkflow

**File:** `backend/internal/api/handlers/runs.go`

`StopWorkflow` is a stub that returns 204 without doing anything. Phase 2 needs:
- Track the active `context.CancelFunc` per workflow ID in a registry
- Call cancel on stop, which propagates to the runner goroutine
- Update run status to `"stopped"` in the DB

---

## Medium priority

### 4. x402 actual payment signing

**File:** `backend/internal/engine/nodes/tool402.go`

`ExecuteTool402` detects a 402 response and parses the payment quote, but the Algorand payment is not sent. The microAlgo amount is calculated but commented out with `// actual signing deferred to Phase 2`.

Needs:
- `wallet.Service` injected into the runner so node executors can call `SignAndSendPayment`
- After payment, retry the original request with a `X-Payment` header containing the signed txn
- Handle payment failure gracefully

### 5. Run replay / history

**File:** `backend/internal/api/router.go`

No endpoint to list past runs for a workflow. Add:
- `GET /workflows/:id/runs` — list runs with status, timestamps, duration
- Frontend RunHistory panel (not yet designed)

### 6. Spend caps per workflow

**Files:** `backend/internal/db/migrations/`, `backend/internal/engine/runner.go`

Phase 1 has no spending limits. Phase 2 needs:
- `spend_cap_algo` column on `workflows` table
- Runner checks cumulative spend before each x402 payment node
- Returns error and stops run if cap exceeded

---

## Low priority / Phase 3

### 7. Cron schedule triggers

Trigger a workflow on a schedule (e.g. every 5 minutes). Needs:
- `schedule` field on workflow (cron expression)
- `robfig/cron/v3` wired into main.go
- Cron job calls `startRun` with `triggeredBy = "cron"`

### 8. Webhook delivery confirmation

`PublicTrigger` (`POST /run/:workflowId`) currently fires the run and returns the `runId` immediately. Phase 2 should optionally support synchronous mode — wait for run completion and return the final output.

### 9. Frontend: workflow validation before deploy

Before calling `POST /workflows/:id/deploy`, validate that the graph is a valid DAG with at least one trigger node and one agent node. Currently the backend runs migrations on invalid graphs and returns partial results.

### 10. Rate limiting

No rate limiting on any endpoint. Before public launch add per-IP and per-user limits on the run trigger and fund endpoints especially.

---

## Known technical debt

| Location | Issue |
|----------|-------|
| `backend/internal/api/handlers/auth.go` | Phase 1 auth always returns `dev-token` — must be replaced before multi-user launch |
| `backend/internal/engine/nodes/tool.go` | `evalMath` uses `go/types.Eval` — limited expression support; swap for `github.com/antonmedv/expr` if complex expressions are needed |
| `frontend/src/components/canvas/LogDrawer.tsx` | Hardcoded mock log lines — not connected to real SSE stream |
| `backend/internal/api/handlers/runs.go` | `StopWorkflow` is a no-op stub |
