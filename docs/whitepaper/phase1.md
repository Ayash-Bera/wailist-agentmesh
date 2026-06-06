# AgentMesh Backend — Phase 1 Design

**Date:** 2026-06-05
**Scope:** Working core — Workflow CRUD, Deploy, Run Engine, SSE Logs, Algorand Wallets, x402 Payments
**Deferred to Phase 2:** Webhook delivery, run replay, spend caps, cron schedule triggers, real JWT auth

---

## 1. Tech Stack

| Concern | Choice | Reason |
|---------|--------|--------|
| Language | Go | Goroutines map to the run engine; explicit error handling; single binary deploy |
| Router | `go-chi/chi/v5` | Lightweight, idiomatic, middleware composable |
| DB driver | `pgx/v5` | Direct SQL, no ORM overhead, excellent PostgreSQL support |
| Migrations | `golang-migrate/migrate/v4` | Numbered `.sql` files, CLI + embedded runner |
| Algorand | `algorand/go-algorand-sdk/v2` | Official SDK; wallet creation, balance, payment signing |
| Cron | `robfig/cron/v3` | Schedule trigger support (Phase 1 stub, Phase 2 active) |
| Config | `joho/godotenv` + `os.Getenv` | `.env` for local, Railway env vars in production |
| Deploy | Railway + Dockerfile | Single binary, PostgreSQL add-on in same Railway project |

---

## 2. Directory Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              ← wire DB, router, SSE broker, start HTTP server
├── internal/
│   ├── api/
│   │   ├── router.go            ← Chi route registration
│   │   ├── middleware.go        ← CORS, auth stub, request logger
│   │   └── handlers/
│   │       ├── auth.go          ← stub: signup/signin/signout return dev token
│   │       ├── workflows.go     ← CRUD: list, get, create, update, delete
│   │       ├── deploy.go        ← provision Algorand wallets, set deployed status
│   │       ├── runs.go          ← trigger run, get status/logs, SSE stream, stop
│   │       └── tools.go         ← x402 quote endpoint
│   ├── engine/
│   │   ├── runner.go            ← load graph, topological sort, goroutine dispatch
│   │   └── nodes/
│   │       ├── provider.go      ← LLM REST calls (OpenAI, Anthropic, Gemini, Mistral, Groq)
│   │       ├── tool.go          ← http_request, calculator, datetime
│   │       ├── tool402.go       ← 402 detect → parse → pay → retry
│   │       └── action.go        ← webhook_post, log, email (Resend)
│   ├── wallet/
│   │   └── algorand.go          ← GenerateWallet, Balance, FundFromDispenser
│   ├── sse/
│   │   └── broker.go            ← per-run channel hub; clients subscribe/unsubscribe
│   ├── db/
│   │   ├── migrations/          ← 001_init.up.sql, 001_init.down.sql, ...
│   │   └── store.go             ← all DB query functions (pgx/v5 rows.Scan)
│   └── models/
│       └── types.go             ← Go structs mirroring src/lib/types.ts
├── Dockerfile
├── .env.example
└── go.mod
```

---

## 3. Database Schema

Single migration file: `001_init.up.sql`

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Auth (stub; userId = 'dev' on all records until real auth ships)
CREATE TABLE users (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email        TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Workflow graphs
CREATE TABLE workflows (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL DEFAULT 'dev',
    name         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'draft',   -- draft | deployed | error
    graph        JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
    deployed_at  TIMESTAMPTZ,
    run_endpoint TEXT,
    notify_url   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One wallet per agent node per deployed workflow
CREATE TABLE agent_wallets (
    id                 TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    workflow_id        TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    agent_node_id      TEXT NOT NULL,
    address            TEXT NOT NULL,
    encrypted_mnemonic TEXT NOT NULL,
    network            TEXT NOT NULL DEFAULT 'testnet',
    UNIQUE (workflow_id, agent_node_id)
);

-- Per-user API keys for LLM providers and tools
CREATE TABLE tool_credentials (
    id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id          TEXT NOT NULL DEFAULT 'dev',
    provider         TEXT NOT NULL,   -- openai | anthropic | gemini | mistral | groq | openweather
    encrypted_api_key TEXT NOT NULL,
    UNIQUE (user_id, provider)
);

-- Workflow runs
CREATE TABLE runs (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    workflow_id  TEXT NOT NULL REFERENCES workflows(id),
    triggered_by TEXT NOT NULL DEFAULT 'manual',   -- manual | chat | webhook | schedule
    status       TEXT NOT NULL DEFAULT 'running',  -- running | success | failed | aborted | stopped
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at  TIMESTAMPTZ,
    input_context JSONB
);

-- Step-level log entries for a run
CREATE TABLE run_logs (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    run_id      TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    step_index  INT  NOT NULL,
    node_id     TEXT NOT NULL,
    node_type   TEXT NOT NULL,
    status      TEXT NOT NULL,   -- pending | running | success | failed
    input       JSONB,
    output      JSONB,
    duration_ms INT,
    ts          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON run_logs (run_id, step_index);
CREATE INDEX ON runs (workflow_id);
```

---

## 4. API Endpoints (Phase 1)

All routes are prefixed with no version segment for now (`/auth/...`, `/workflows/...`).
Auth middleware reads `Authorization: Bearer <token>` but in Phase 1 always sets `userID = "dev"` and never rejects.

### Auth (stubs)

```
POST  /auth/signup    body: { email, password, org }   → { token: "dev-token" }
POST  /auth/signin    body: { email, password }        → { token: "dev-token" }
POST  /auth/signout                                    → 204
GET   /auth/me                                         → { id: "dev", email }
```

### Workflows

```
GET    /workflows                → []Workflow
POST   /workflows                body: { name }                    → Workflow
GET    /workflows/:id            → Workflow (graph + status)
PUT    /workflows/:id            body: Partial<Workflow>           → Workflow
DELETE /workflows/:id                                             → 204
```

### Deploy

```
POST   /workflows/:id/deploy
  → provisions one Algorand wallet per agent node in the graph
  → sets workflow.status = "deployed", workflow.run_endpoint
  → returns { workflowId, status, runEndpoint, agents: [{nodeId, address, network}], deployedAt }

GET    /workflows/:id/agents/:agentId/balance
  → hits Algod REST, returns { address, balance, network }

POST   /workflows/:id/agents/:agentId/fund
  body: { amount }
  → calls testnet dispenser, returns { txHash, balance }
```

### Runs

```
POST   /run/:workflowId          public curl-able trigger (chat, webhook, manual)
  body: any JSON (becomes input_context on the run)
  → starts run asynchronously, returns { runId }

POST   /workflows/:id/run        same as above, frontend-facing alias
POST   /workflows/:id/stop       → marks run stopped, closes SSE channel

GET    /runs/:runId              → { run, logs: []RunLog }
GET    /runs/:runId/stream       SSE stream of LogEvent lines
  event: log
  data: { stepIndex, nodeId, nodeType, status, output, durationMs, ts }
```

### Tools

```
POST   /tools/x402/quote
  body: { url }
  → GET url, read 402 headers, return { price, unit, network, recipient }
  → does NOT pay
```

---

## 5. Run Engine

### 5.1 Graph representation

The workflow graph stored in `workflows.graph` is the same JSON shape as `src/lib/types.ts`:

```go
type NodeType string  // "trigger" | "agent" | "provider" | "tool" | "tool402" | "action" | "end"
type EdgeKind string  // "flow" | "attach"

type WorkflowNode struct {
    ID           string   `json:"id"`
    Type         NodeType `json:"type"`
    Template     string   `json:"template,omitempty"`
    Name         string   `json:"name,omitempty"`
    // provider-specific
    APIKey       string   `json:"apiKey,omitempty"`
    Model        string   `json:"model,omitempty"`
    SystemPrompt string   `json:"systemPrompt,omitempty"`
    // tool-specific
    URL          string   `json:"url,omitempty"`
    Method       string   `json:"method,omitempty"`
    // tool402-specific
    Endpoint     string   `json:"endpoint,omitempty"`
    Provider     string   `json:"provider,omitempty"`
    // trigger-specific
    Source       string   `json:"source,omitempty"`
}

type WorkflowEdge struct {
    ID     string   `json:"id"`
    From   string   `json:"from"`
    To     string   `json:"to"`
    Kind   EdgeKind `json:"kind"`
    ToPort string   `json:"toPort,omitempty"`
}
```

### 5.2 Execution algorithm

```
func RunWorkflow(ctx, graph, inputContext, runID):

  1. Separate edges:
       flowEdges   = edges where kind == "flow"
       attachEdges = edges where kind == "attach"  (model→agent, tools→agent)

  2. Build attachMap: agentNodeID → { model: providerNode, tools: []toolNode }
     by walking attachEdges.

  3. Topological sort on flowEdges only (Kahn's algorithm).
     Result: ordered [][]WorkflowNode where inner slices are parallel groups
     (nodes with no inter-dependency within the same depth can run concurrently).

  4. runCtx = RunContext{ inputs: inputContext, outputs: map[nodeID]any{}, mu: sync.RWMutex }

  5. For each parallel group:
       wg := sync.WaitGroup
       for each node in group:
           wg.Add(1)
           go func(node):
               defer wg.Done()
               WriteLog(runID, node, status=running)
               result, err := ExecuteNode(ctx, node, attachMap, runCtx)
               if err:
                   WriteLog(runID, node, status=failed, output=err)
                   cancelRun()
                   return
               runCtx.Set(node.ID, result)
               WriteLog(runID, node, status=success, output=result)
               broker.Publish(runID, LogEvent{...})
       wg.Wait()
       if cancelled: break

  6. UpdateRunStatus(runID, success|failed|stopped)
     broker.Close(runID)
```

### 5.3 Node executors

**Provider (LLM)**

Calls provider REST APIs directly. No vendor SDKs — just `net/http`. Each provider:

| Provider | Base URL | Auth header |
|----------|----------|-------------|
| OpenAI | `https://api.openai.com/v1/chat/completions` | `Authorization: Bearer` |
| Anthropic | `https://api.anthropic.com/v1/messages` | `x-api-key` + `anthropic-version` |
| Gemini | `https://generativelanguage.googleapis.com/v1beta/models/:model:generateContent` | `?key=` |
| Mistral | `https://api.mistral.ai/v1/chat/completions` | `Authorization: Bearer` |
| Groq | `https://api.groq.com/openai/v1/chat/completions` | `Authorization: Bearer` (OpenAI-compatible) |

API key source: node's `apiKey` field (set in frontend Inspector), falling back to `tool_credentials` table for the user.

**Tool — built-in**

| Template | What it does |
|----------|-------------|
| `http` | Configurable GET/POST, headers, body from runCtx interpolation |
| `calc` | `strconv` + `go-expr` or manual eval — no external API |
| `datetime` | `time.Now().UTC()` formatted as RFC3339 |

**Tool402 — x402 payment flow**

```
1. GET node.Endpoint
2. If status != 402: return body directly (free endpoint)
3. Parse X-Payment-Required header:
     { amount, unit, network, recipient, ... }
4. Load agent wallet (decrypt mnemonic from DB)
5. Build Algorand payment transaction via go-algorand-sdk:
     - Connect to Algod (testnet: https://testnet-api.algonode.cloud)
     - GetSuggestedParams()
     - MakePaymentTxn(from=wallet.address, to=recipient, amount=price)
     - Sign with private key
     - BroadcastTransaction()
     - WaitForConfirmation(txID)
6. Retry original request with header:
     X-Payment: <base64-encoded proof or txID depending on x402 spec version>
7. Return response body + { pricePaid, txHash } appended to runCtx
```

**Action**

| Template | What it does |
|----------|-------------|
| `webhook` / `post_webhook` | POST runCtx output as JSON to configured URL |
| `log` | Write to run_logs only, no external call |
| `email` | POST to Resend API (`https://api.resend.com/emails`) |

**End node** — writes final output to run, completes the run.

### 5.4 SSE broker

```go
type Broker struct {
    mu      sync.Mutex
    runs    map[string]*runHub  // runID → hub
}

type runHub struct {
    events  chan LogEvent
    clients map[chan LogEvent]struct{}
    done    chan struct{}
}

// Publisher (engine) calls:
broker.Publish(runID, event)

// HTTP handler (SSE stream endpoint):
ch := broker.Subscribe(runID)
defer broker.Unsubscribe(runID, ch)
w.Header().Set("Content-Type", "text/event-stream")
for {
    select {
    case ev := <-ch:
        fmt.Fprintf(w, "event: log\ndata: %s\n\n", marshal(ev))
        flusher.Flush()
    case <-hub.done:
        fmt.Fprintf(w, "event: done\ndata: {}\n\n")
        flusher.Flush()
        return
    case <-r.Context().Done():
        return
    }
}
```

---

## 6. Algorand Wallet Service

```go
// wallet/algorand.go

const AlgodURL  = "https://testnet-api.algonode.cloud"
const AlgodToken = ""  // algonode doesn't require a token

func GenerateWallet() (address, mnemonic string, err error)
  // crypto.GenerateAccount() → mnemonic.FromPrivateKey()

func Balance(address string) (microAlgo uint64, err error)
  // algodClient.AccountInformation(address)

func FundFromDispenser(address string, amount uint64) (txHash string, err error)
  // POST https://dispenser.testnet.aws.algodev.network/?receiver=<address>&amount=<amount>

func SignAndSendPayment(mnemonic, to string, microAlgo uint64) (txHash string, err error)
  // mnemonic.ToPrivateKey() → MakePaymentTxn → SignTransaction → SendRawTransaction
```

Mnemonic encryption at rest: AES-256-GCM, key from `ENCRYPTION_KEY` env var. Simple `encrypt(plaintext, key) []byte` / `decrypt(ciphertext, key) string` helpers in `wallet/crypto.go`.

---

## 7. Environment Variables

```bash
# backend/.env.example

DATABASE_URL=postgres://user:pass@host:5432/agentmesh?sslmode=require
PORT=8080
ENCRYPTION_KEY=32-byte-hex-key-for-wallet-mnemonics

# Algorand
ALGOD_URL=https://testnet-api.algonode.cloud
ALGOD_TOKEN=
ALGORAND_NETWORK=testnet

# Optional: platform-level LLM keys (fallback if user hasn't set node-level keys)
OPENAI_API_KEY=
ANTHROPIC_API_KEY=

# CORS origin (Next.js frontend)
CORS_ORIGIN=http://localhost:3000
```

---

## 8. Dockerfile

```dockerfile
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /app/server .
EXPOSE 8080
CMD ["./server"]
```

---

## 9. Go Module Dependencies

```
github.com/go-chi/chi/v5
github.com/jackc/pgx/v5
github.com/golang-migrate/migrate/v4
github.com/golang-migrate/migrate/v4/database/pgx/v5
github.com/golang-migrate/migrate/v4/source/file
github.com/algorand/go-algorand-sdk/v2
github.com/joho/godotenv
github.com/robfig/cron/v3
github.com/google/uuid
golang.org/x/crypto
```

---

## 10. Phase 2 Scope (not in this plan)

| Module | What it adds |
|--------|-------------|
| Webhook delivery | Async outbound POST on run complete, 3-retry exponential backoff |
| Run replay | `POST /runs/:runId/replay` — re-execute with stored inputContext |
| Spend caps | Pre-step ALGO + token budget check, 429 on maxRunsPerDay |
| Schedule triggers | Active cron registration from trigger node config |
| Real auth | JWT RS256, `POST /auth/signup` creates real user row, middleware validates |

---

## 11. Frontend Integration Checklist

When the backend is running, connect the frontend by:

1. Set `NEXT_PUBLIC_API_URL=http://localhost:8080` in `.env.local`
2. The `agentmesh_token` localStorage key must be sent as `Authorization: Bearer <token>` — already wired in `src/lib/api.ts`
3. Verify endpoint shapes match the stubs in `src/lib/api.ts` — any field rename in the backend response needs a matching change in the frontend types (`src/lib/types.ts`)
4. SSE stream: frontend should connect `EventSource` to `GET /runs/:runId/stream` after starting a run — the LogDrawer in `src/components/canvas/LogDrawer.tsx` needs to subscribe to this

---

## 12. Implementation Order

1. `go mod init` + deps + `cmd/server/main.go` skeleton
2. DB connection + migrations (`001_init.up.sql`)
3. `db/store.go` — all query functions (no logic, just DB)
4. Models (`models/types.go`)
5. Chi router + CORS + auth stub middleware
6. Workflow CRUD handlers
7. Wallet service (`wallet/algorand.go`)
8. Deploy handler (wallet creation per agent node)
9. SSE broker (`sse/broker.go`)
10. Run engine — runner + topological sort (no node execution yet, just skeleton)
11. Provider node executor (OpenAI first, test end-to-end)
12. Tool node executor (http_request, calculator, datetime)
13. x402 node executor + quote endpoint
14. Action node executor (webhook_post, log)
15. Wire balance + fund endpoints
16. Dockerfile + Railway deploy
17. Smoke test: frontend pointing at real backend, run a workflow end-to-end
