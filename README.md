# AgentMesh

No-code platform for building autonomous agent workflows with Algorand wallets and x402 micropayments.

## Structure

```
frontend/   — Next.js 16 app (React 19, Tailwind 4, TypeScript)
backend/    — Go HTTP server (chi, pgx/v5, Algorand SDK v2)
docs/       — specs, plans, whitepaper
```

## Prerequisites

- Node.js 20+
- Go 1.23+
- PostgreSQL (local or Railway)

---

## Local development

### 1. Backend

```bash
cp backend/.env.example backend/.env
```

Edit `backend/.env`:

```bash
DATABASE_URL=postgres://postgres:password@localhost:5432/agentmesh?sslmode=disable
ENCRYPTION_KEY=abcdefghijklmnopqrstuvwxyz123456   # exactly 32 chars
PORT=8080
BASE_URL=http://localhost:8080
CORS_ORIGIN=http://localhost:3000

# optional — needed for AI nodes
OPENAI_API_KEY=
ANTHROPIC_API_KEY=

# Algorand testnet (defaults work without a token)
ALGOD_URL=https://testnet-api.algonode.cloud
ALGOD_TOKEN=
ALGORAND_NETWORK=testnet
```

Create the database, then start the server:

```bash
createdb agentmesh
cd backend && go run ./cmd/server
# → AgentMesh backend listening on :8080
```

Migrations run automatically on startup.

Verify:
```bash
curl http://localhost:8080/health
# → ok
```

### 2. Frontend

```bash
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" > frontend/.env.local
cd frontend && npm install && npm run dev
# → localhost:3000
```

---

## Railway deployment (recommended)

Railway handles PostgreSQL and auto-detects the Dockerfile.

```bash
# install Railway CLI once
npm i -g @railway/cli

railway login
cd backend && railway init
railway add --plugin postgresql        # provisions managed Postgres + sets DATABASE_URL

railway variables set ENCRYPTION_KEY=<32-char-string>
railway variables set CORS_ORIGIN=http://localhost:3000
railway variables set BASE_URL=<your-railway-backend-url>

railway up
```

Once deployed, grab the public Railway URL and point the frontend at it:

```bash
echo "NEXT_PUBLIC_API_URL=https://<your-app>.railway.app" > frontend/.env.local
cd frontend && npm run dev
```

---

## Backend API

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/health` | Health check |
| POST | `/auth/signin` | Sign in (Phase 1: returns dev token) |
| POST | `/auth/signup` | Sign up (Phase 1: returns dev token) |
| GET | `/workflows` | List workflows |
| POST | `/workflows` | Create workflow |
| GET | `/workflows/:id` | Get workflow |
| PUT | `/workflows/:id` | Update workflow |
| DELETE | `/workflows/:id` | Delete workflow |
| POST | `/workflows/:id/deploy` | Provision Algorand wallets per agent node |
| GET | `/workflows/:id/agents/:agentId/balance` | Agent wallet balance |
| POST | `/workflows/:id/agents/:agentId/fund` | Fund agent from testnet dispenser |
| POST | `/workflows/:id/run` | Trigger a run |
| POST | `/workflows/:id/stop` | Stop workflow |
| GET | `/runs/:runId` | Get run + logs |
| GET | `/runs/:runId/stream` | SSE log stream |
| POST | `/run/:workflowId` | Public webhook trigger |
| POST | `/tools/x402/quote` | Quote an x402 endpoint |

---

## Frontend commands

```bash
cd frontend
npm run dev      # dev server (localhost:3000)
npm run build    # production build
npm run lint     # eslint
```

## Backend commands

```bash
cd backend
go run ./cmd/server   # run server
go build ./...        # build binary
go test ./...         # run tests
```

---

## Notes

- Auth is a Phase 1 stub — any credentials work, token is always `dev-token`. See `docs/whats-left.md` for Phase 2 roadmap.
- `NEXT_PUBLIC_API_URL` is the only switch between mock data and real backend. Unset = mock mode.
- Algorand wallets use testnet by default. Set `ALGORAND_NETWORK=mainnet` with a real algod endpoint for production.
