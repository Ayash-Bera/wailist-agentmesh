# Frontend Integration Plan

## Current State (as of 2026-06-08)

### Backend — 100% Real
All backend features are fully implemented and tested. No stubs remain.

### Frontend — Auth only. Everything else is mocked.
Only `AuthPage.tsx`, `useAuth.ts`, and `LandingPage.tsx` (waitlist) import and use `api.ts`.
`WorkflowsPage` and `CanvasPage` never import `api.ts` at all.

---

## What's Left to Integrate (in order)

---

### ~~Task 1 — WorkflowsPage: load real workflows~~ ✅ DONE

**File:** `src/components/workflows/WorkflowsPage.tsx`

What to change:
- Remove `import { WORKFLOWS } from "@/lib/data"`
- On mount, call `workflows.list()` from `api.ts` and store result in state
- Show a loading skeleton while fetching
- "New workflow" button: call `workflows.create("Untitled workflow")`, then `router.push("/workflows/" + id)`
- KPI card "Active workflows": derive count from real list
- KPI cards "agents deployed / ALGO spent / runs": hide or leave hardcoded for now (no aggregate endpoint yet)

Backend endpoint: `GET /workflows` → `POST /workflows`

---

### ~~Task 2 — CanvasPage: load real workflow~~ ✅ DONE

**File:** `src/components/canvas/CanvasPage.tsx`

What to change:
- Remove `import { SAMPLE_WORKFLOW } from "@/lib/data"`
- On mount, call `workflows.get(workflowId)` and set as initial state
- Handle the `/workflows/new` route: call `workflows.create("Untitled workflow")` first, then `router.replace("/workflows/" + newId)` so the URL reflects the real ID
- Show loading state before data arrives

Backend endpoint: `GET /workflows/:id`

---

### ~~Task 3 — CanvasPage: auto-save~~ ✅ DONE

**File:** `src/components/canvas/CanvasPage.tsx`

What to change:
- Debounce `workflows.update(id, { name, nodes, edges })` whenever nodes, edges, or workflow name changes (~1.5–2s delay)
- Replace the hardcoded `"auto-saved · 12s ago"` string in the topbar with the real timestamp from the last successful save
- Show "saving…" during the debounce window

Backend endpoint: `PUT /workflows/:id`

---

### ~~Task 4 — Deploy: call real backend~~ ✅ DONE

**File:** `src/components/canvas/CanvasPage.tsx`

What to change:
- `onDeploy`: replace the `Math.random()` wallet generation with a call to `workflows.deploy(id)`
- Backend returns `{ agents: [{ nodeId, address, network }] }`
- For each agent in the response, update the matching node's `wallet` field in state so the Inspector can display the real Algorand address
- Set `deployed = true` after success

Backend endpoint: `POST /workflows/:id/deploy`

---

### ~~Task 5 — Inspector: show real wallet address and balance~~ ✅ DONE

**File:** `src/components/canvas/Inspector.tsx`

What to change:
- When a selected node has a `wallet.address`, display the real Algorand testnet address
- Add a "Refresh balance" button that calls `GET /workflows/:id/agents/:agentId/balance`
- `onFund`: call `agents.fund(workflowId, agentId, amount)` from `api.ts` instead of updating local state
- Refresh balance after fund completes

Backend endpoints: `GET /workflows/:id/agents/:agentId/balance` → `POST /workflows/:id/agents/:agentId/fund`

---

### ~~Task 6 — Run: call real backend + connect SSE log stream~~ ✅ DONE

**Files:** `src/components/canvas/CanvasPage.tsx`, `src/components/canvas/LogDrawer.tsx`

What to change in CanvasPage:
- `onRun` (start): call `workflows.run(id)` → get real `runId` back → store in state → pass to `LogDrawer`
- `onRun` (stop): call `workflows.stop(id)`

What to change in LogDrawer:
- Remove `import { LOG_LINES } from "@/lib/data"`
- When `runId` is provided, open an `EventSource` to `GET /runs/:runId/stream`
- Parse `event: log` payloads (shape: `{ nodeId, nodeType, status, output, durationMs, ts }`) and render as real log lines
- On `event: done`, mark run complete and close the stream
- On component unmount or stop, close the `EventSource`

Backend endpoints: `POST /workflows/:id/run` → `GET /runs/:runId/stream`

---

### ~~Task 7 — Auth token validity check (polish)~~ ✅ DONE

**File:** `src/hooks/useAuth.ts`

What to change:
- On app load, if a token exists in localStorage, call `auth.me()` to verify it's still valid
- If the request fails (401/expired), clear localStorage and redirect to `/signin`
- This prevents a stale token from leaving the user in a broken logged-in state

Backend endpoint: `GET /auth/me`

---

### ~~Task 8 — Error handling (polish)~~ ✅ DONE

**Files:** all components above

What to change:
- Wrap all API calls with try/catch
- Show a toast on failure (a `<Toast>` component already exists in `src/components/ui/index.tsx`)
- Specific cases: load failure → show error + retry button; save failure → show toast; deploy failure → show error in topbar; run failure → show in LogDrawer

---

## Environment Setup

```bash
# frontend/.env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
```

The backend reads from its own `.env` (already configured via docker-compose).

---

## Integration Order

Do tasks in order — each one builds on the previous:

```
1 → WorkflowsPage loads real workflows
2 → Canvas loads real workflow by ID
3 → Canvas saves changes back to DB
4 → Deploy creates real Algorand wallets
5 → Inspector shows real wallet address + balance
6 → Run triggers real execution + LogDrawer streams live logs
7 → Auth token validation
8 → Error handling
```
