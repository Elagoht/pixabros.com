# Admin SPA Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the React + Vite admin SPA *shell*: project scaffold, login page, a minimal authenticated shell/dashboard, a change-password screen, logout, and a Service Worker for app-shell asset caching — served by Plan A/B's Go backend at `/I-am-a-pixabro/*`. Adds the one missing backend primitive (`GET /api/admin/whoami`) the SPA needs to restore auth state after a page refresh, since the session cookie is `HttpOnly` and unreadable from JS.

**Architecture:** A TypeScript React SPA (Vite-built), talking to the existing `/api/admin/*` JSON endpoints over `fetch` with `credentials: "include"`. Auth state lives in a React context that resolves once on mount by calling `whoami`. `react-router-dom` handles the (currently tiny) route tree so later per-module CRUD screens slot in without restructuring. A hand-written Service Worker caches only the built app-shell assets (cache-first) and never touches `/api/*`.

**Architecture Decisions:**

1. **TypeScript, not plain JS.** The user's global CLAUDE.md rule bans `any` everywhere, including TS. Adopting TypeScript makes that rule enforceable by the compiler and linter instead of by discipline alone — the exact discipline this codebase already has on the Go side (no `any`/`interface{}` misuse).
2. **Auth state via a `whoami` probe.** Because `pixabros_session` is `HttpOnly`, no client-side check can tell the SPA "am I logged in?" after a refresh. Plan A's `internal/adminapi` already has the session-validating middleware (`RequireSession`) and everything `Whoami` needs (`AdminRepo.FindByID`, `AdminIDFromContext`) — adding a tiny `GET /api/admin/whoami` handler that reuses them is far simpler than any client-only alternative (e.g. a non-HttpOnly marker cookie, which would leak auth-state to JS unnecessarily).
3. **`react-router-dom`, not conditional rendering.** A shell with 3 screens *could* be done with `if`/`else`, but every future per-module plan (Games, Members, Devlog, Awards, Contact, Site Settings, Media, regen-job status) needs its own URL for deep-linking and bookmarking inside the admin panel. Wiring `react-router-dom` now — even for 3 routes — means those plans add `<Route>` entries instead of restructuring the shell.
4. **Hand-written Service Worker, not `vite-plugin-pwa`.** The caching rule is simple and fixed for the life of this shell: cache-first for built JS/CSS/HTML under `/I-am-a-pixabro/`, network-only (never touched) for `/api/*`. A 20-line hand-written `public/sw.js` makes that rule auditable in one file; `vite-plugin-pwa`'s generated Workbox config is a black box that would need `workbox-build` tuning to guarantee `/api/*` is never intercepted — extra dependency and indirection for no benefit at this scope.
5. **Vitest + React Testing Library, real assertions.** Every component gets rendered with RTL and asserted against real DOM output (`getByText`, `getByRole`, form submission via `fireEvent`) — no snapshot placeholders. The typed API client is tested by mocking `global.fetch` and asserting on the parsed `ApiResult` shape. `AuthContext` is tested by mocking the API client module and asserting the exposed `status`/`username` transitions. The Service Worker's *behavioral* logic (which requests to intercept) is extracted into a plain, unit-testable function; the SW file itself is verified manually in a browser, since `jsdom` has no `ServiceWorkerGlobalScope`.

**Tech Stack:** Go 1.22+ (whoami addition only), Vite 5, React 18, TypeScript 5 (strict), `react-router-dom` 6, Vitest 2 + `@testing-library/react` + `@testing-library/jest-dom`, ESLint 8 + `@typescript-eslint` (with `no-explicit-any` as an error).

**Depends on:** `docs/superpowers/plans/2026-08-10-backend-core-data-model.md` (Plan A) and `docs/superpowers/plans/2026-08-10-content-rendering-pipelines.md` (Plan B) — this plan assumes both already exist and their tests pass, and it modifies the **post-Plan-B** state of `internal/httpserver/router.go` (`Dependencies{Admins, Sessions, Store, Files, AdminUIDir, PlayDir, AssetsDir}`) and relies on, but does not modify, `cmd/server/main.go`.

## Global Constraints

- Never use TypeScript's `any` — use `unknown` + type guards, or precise generics, everywhere (user's global CLAUDE.md rule; enforced here via `@typescript-eslint/no-explicit-any: "error"`). Never use Go's `any` alias either (unchanged from Plan A/B).
- Session cookie contract is fixed by Plan A: name `pixabros_session`, `HttpOnly, Secure, SameSite=Strict`. The SPA never reads it directly.
- Every API response follows `{"error": {"code": "...", "message": "..."}}` on failure; the client's type guards must not assume any other shape.
- The SPA is mounted at `/I-am-a-pixabro/*` only — Vite's `base` and `react-router-dom`'s `basename` must both reflect that exact path.
- The Service Worker's scope is `/I-am-a-pixabro/` (its own registration path); it must never intercept `/api/*` requests.
- Per-module CRUD screens (Games, Members, Devlog, Awards, Contact inbox, Homepage/Site Settings, Media library, regen-job status) are **out of scope** — their backend APIs don't exist yet and land in later per-module plans.
- Git commits in this repo: self-committed, one-sentence semantic messages, no co-author trailer.

## Scope

This plan (Plan C of three) builds only the admin SPA shell: scaffold, login, minimal authenticated dashboard, change-password, logout, Service Worker, and the one backend addition (`whoami`) the shell needs. Out of scope, deferred to future per-module plans:
- Any Games/Members/Devlog/Awards/Contact/Site-Settings/Media/regen-job CRUD screens or their backend REST endpoints.
- Contact-form honeypot/rate-limiting (that's public-site scope, not admin).
- Public-site MPA rendering (separate, already covered by Plan B).

---

## File Structure

```
internal/
  adminapi/
    handlers.go          # (modify) add Whoami handler + whoamiResponse
    handlers_test.go      # (modify) add TestWhoami_Success, TestWhoami_Unauthorized
  httpserver/
    router.go              # (modify) mount GET /api/admin/whoami
    router_test.go          # (modify) assert whoami route reachable after login
admin-ui/
  package.json
  tsconfig.json
  vite.config.ts
  .eslintrc.cjs
  .gitignore
  index.html
  public/
    sw.js
  src/
    vite-env.d.ts
    main.tsx
    App.tsx
    App.test.tsx
    index.css
    setupTests.ts
    registerServiceWorker.ts
    registerServiceWorker.test.ts
    api/
      client.ts
      client.test.ts
    auth/
      AuthContext.tsx
      AuthContext.test.tsx
    components/
      ProtectedRoute.tsx
      ProtectedRoute.test.tsx
    pages/
      LoginPage.tsx
      LoginPage.test.tsx
      ChangePasswordPage.tsx
      ChangePasswordPage.test.tsx
      Shell.tsx
      Shell.test.tsx
      DashboardPage.tsx
Makefile               # (create) admin-build target
.gitignore              # (modify) ignore admin-ui/node_modules
```

Each `admin-ui/src/*` directory owns one concern (typed API access, auth state, route guarding, screens). `internal/adminapi` and `internal/httpserver` get small, additive modifications — no existing Plan A/B behavior changes.

---

### Task 1: Backend `whoami` endpoint

**Files:**
- Modify: `internal/adminapi/handlers.go`
- Modify: `internal/adminapi/handlers_test.go`
- Modify: `internal/httpserver/router.go`
- Modify: `internal/httpserver/router_test.go`

**Interfaces:**
- Consumes: `auth.AdminRepo.FindByID` (Plan A Task 6), `adminapi.RequireSession`, `adminapi.AdminIDFromContext` (Plan A Task 9)
- Produces: `(*AuthHandlers).Whoami` (`http.HandlerFunc`-shaped), mounted as `GET /api/admin/whoami` behind `RequireSession`

- [ ] **Step 1: Write the failing tests for the `Whoami` handler**

Add to `internal/adminapi/handlers_test.go` (append; the file already has `package adminapi`, the same imports, and the `setupHandlers` helper from Plan A Task 9 — no new imports needed):

```go
func TestWhoami_Success(t *testing.T) {
	handlers, sessions, _, adminID := setupHandlers(t)
	token, _, err := sessions.Create(adminID)
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/whoami", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()

	RequireSession(sessions, handlers.Whoami)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Username != "furkan" {
		t.Errorf("username = %q, want %q", body.Username, "furkan")
	}
}

func TestWhoami_Unauthorized(t *testing.T) {
	handlers, sessions, _, _ := setupHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/whoami", nil)
	rec := httptest.NewRecorder()

	RequireSession(sessions, handlers.Whoami)(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adminapi/... -v -run TestWhoami`
Expected: FAIL — `(*AuthHandlers).Whoami` undefined.

- [ ] **Step 3: Implement the `Whoami` handler**

Add to `internal/adminapi/handlers.go` (append; same file, same imports as Plan A Task 9):

```go
type whoamiResponse struct {
	Username string `json:"username"`
}

func (h *AuthHandlers) Whoami(w http.ResponseWriter, r *http.Request) {
	adminID, ok := AdminIDFromContext(r.Context())
	if !ok {
		httpapi.WriteError(w, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}

	admin, err := h.admins.FindByID(adminID)
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "could not load admin")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, whoamiResponse{Username: admin.Username})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adminapi/... -v -run TestWhoami`
Expected: PASS

- [ ] **Step 5: Wire the route into the router**

Modify `internal/httpserver/router.go` (post-Plan-B state): add the whoami route next to the other `/api/admin/*` routes:

```go
	authHandlers := adminapi.NewAuthHandlers(deps.Admins, deps.Sessions)
	mux.HandleFunc("POST /api/admin/login", authHandlers.Login)
	mux.HandleFunc("POST /api/admin/logout", authHandlers.Logout)
	mux.HandleFunc("POST /api/admin/change-password", adminapi.RequireSession(deps.Sessions, authHandlers.ChangePassword))
	mux.HandleFunc("GET /api/admin/whoami", adminapi.RequireSession(deps.Sessions, authHandlers.Whoami))
```

- [ ] **Step 6: Write the failing router-level test**

Add to `internal/httpserver/router_test.go` (append inside `TestRouter_LoginAndSingleOriginServing`, right after the existing login assertions and before the `adminResp` check):

```go
	client := &http.Client{}
	cookieReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/admin/whoami", nil)
	for _, c := range loginResp.Cookies() {
		cookieReq.AddCookie(c)
	}
	whoamiResp, err := client.Do(cookieReq)
	if err != nil {
		t.Fatalf("whoami request error = %v", err)
	}
	if whoamiResp.StatusCode != http.StatusOK {
		t.Fatalf("whoami status = %d, want %d", whoamiResp.StatusCode, http.StatusOK)
	}

	anonResp, err := srv.Client().Get(srv.URL + "/api/admin/whoami")
	if err != nil {
		t.Fatalf("anonymous whoami request error = %v", err)
	}
	if anonResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous whoami status = %d, want %d", anonResp.StatusCode, http.StatusUnauthorized)
	}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/httpserver/... -v`
Expected: PASS (fails first if run before Step 5's route registration is in place).

- [ ] **Step 8: Run the full backend test suite**

Run: `go test ./...`
Expected: PASS for every package.

- [ ] **Step 9: Commit**

```bash
git add internal/adminapi internal/httpserver
git commit -m "feat: add admin whoami endpoint for session restoration"
```

---

### Task 2: Vite + React + TypeScript scaffold

**Files:**
- Create: `admin-ui/package.json`
- Create: `admin-ui/tsconfig.json`
- Create: `admin-ui/vite.config.ts`
- Create: `admin-ui/.eslintrc.cjs`
- Create: `admin-ui/.gitignore`
- Create: `admin-ui/index.html`
- Create: `admin-ui/src/vite-env.d.ts`
- Create: `admin-ui/src/index.css`
- Create: `admin-ui/src/setupTests.ts`
- Create: `admin-ui/src/App.tsx`
- Create: `admin-ui/src/App.test.tsx`
- Create: `admin-ui/src/main.tsx` (placeholder; rewritten in Task 7 to add real routing, and again in Task 8 to add SW registration)

**Interfaces:**
- Consumes: nothing from earlier tasks
- Produces: a buildable, testable Vite project — `npm run build`, `npm test`, `npm run lint` all work

- [ ] **Step 1: Create the project directory and `package.json`**

`admin-ui/package.json`:

```json
{
  "name": "admin-ui",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "lint": "eslint . --ext ts,tsx"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.26.2"
  },
  "devDependencies": {
    "@testing-library/jest-dom": "^6.5.0",
    "@testing-library/react": "^16.0.1",
    "@types/node": "^22.7.4",
    "@types/react": "^18.3.10",
    "@types/react-dom": "^18.3.0",
    "@typescript-eslint/eslint-plugin": "^7.16.1",
    "@typescript-eslint/parser": "^7.16.1",
    "@vitejs/plugin-react": "^4.3.2",
    "eslint": "^8.57.1",
    "eslint-plugin-react-hooks": "^4.6.2",
    "eslint-plugin-react-refresh": "^0.4.12",
    "jsdom": "^25.0.1",
    "typescript": "^5.6.2",
    "vite": "^5.4.9",
    "vitest": "^2.1.2"
  }
}
```

- [ ] **Step 2: Install dependencies**

```bash
cd admin-ui
npm install
```

Expected: `node_modules/` and `package-lock.json` are created; no error output.

- [ ] **Step 3: Add `.gitignore`**

`admin-ui/.gitignore`:

```gitignore
node_modules/
dist/
*.local
```

- [ ] **Step 4: Add TypeScript config**

`admin-ui/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "types": ["@testing-library/jest-dom"]
  },
  "include": ["src", "vite.config.ts"]
}
```

- [ ] **Step 5: Add ESLint config with the no-`any` rule**

`admin-ui/.eslintrc.cjs`:

```js
module.exports = {
  root: true,
  env: { browser: true, es2021: true, node: true },
  extends: [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:react-hooks/recommended",
  ],
  parser: "@typescript-eslint/parser",
  parserOptions: { ecmaVersion: "latest", sourceType: "module" },
  plugins: ["react-refresh"],
  rules: {
    "@typescript-eslint/no-explicit-any": "error",
    "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
  },
  ignorePatterns: ["dist", ".eslintrc.cjs", "node_modules"],
};
```

- [ ] **Step 6: Add Vite config (base path, output dir, Vitest settings)**

`admin-ui/vite.config.ts`:

```ts
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "node:path";

const outDir = process.env.ADMIN_UI_OUT_DIR
  ? path.resolve(process.env.ADMIN_UI_OUT_DIR)
  : path.resolve(__dirname, "../data/admin-dist");

export default defineConfig({
  base: "/I-am-a-pixabro/",
  plugins: [react()],
  build: {
    outDir,
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    globals: false,
    setupFiles: ["./src/setupTests.ts"],
  },
});
```

- [ ] **Step 7: Add `index.html`**

`admin-ui/index.html`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Pixabros Admin</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 8: Add `vite-env.d.ts` and `index.css`**

`admin-ui/src/vite-env.d.ts`:

```ts
/// <reference types="vite/client" />
```

`admin-ui/src/index.css`:

```css
:root {
  font-family: system-ui, sans-serif;
  color-scheme: light dark;
}

body {
  margin: 0;
}
```

- [ ] **Step 9: Add the Vitest setup file**

`admin-ui/src/setupTests.ts`:

```ts
import "@testing-library/jest-dom/vitest";
```

- [ ] **Step 10: Write the failing smoke test**

`admin-ui/src/App.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";

describe("App", () => {
  it("renders the admin shell heading", () => {
    render(<App />);
    expect(screen.getByText("Pixabros Admin")).toBeInTheDocument();
  });
});
```

- [ ] **Step 11: Run the test to verify it fails**

```bash
cd admin-ui
npm test
```

Expected: FAIL — `./App` does not exist yet.

- [ ] **Step 12: Implement the placeholder `App` and entrypoint**

`admin-ui/src/App.tsx` (this placeholder is fully replaced in Task 7 with real routing — kept minimal here to prove the toolchain end-to-end):

```tsx
export default function App() {
  return <div>Pixabros Admin</div>;
}
```

`admin-ui/src/main.tsx`:

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./index.css";

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("root element not found");
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
```

- [ ] **Step 13: Run the test to verify it passes**

```bash
npm test
```

Expected: PASS

- [ ] **Step 14: Verify the build and lint pipelines work**

```bash
npm run build
npm run lint
```

Expected: `npm run build` produces `../data/admin-dist/index.html` + a hashed `assets/` bundle; `npm run lint` reports no errors.

- [ ] **Step 15: Commit**

```bash
git add admin-ui
git commit -m "feat: scaffold vite react typescript admin ui project"
```

---

### Task 3: Typed API client

**Files:**
- Create: `admin-ui/src/api/client.ts`
- Create: `admin-ui/src/api/client.test.ts`

**Interfaces:**
- Consumes: the 4 backend endpoints from Task 1 / Plan A (`login`, `logout`, `change-password`, `whoami`)
- Produces: `login`, `logout`, `whoami`, `changePassword` functions; `ApiResult<T>`, `ApiError`, `LoginRequest`, `LoginResponse`, `WhoamiResponse`, `ChangePasswordRequest` types

- [ ] **Step 1: Write the failing tests**

`admin-ui/src/api/client.test.ts`:

```ts
import { afterEach, describe, expect, it, vi } from "vitest";
import { changePassword, login, logout, whoami } from "./client";

function mockFetchOnce(status: number, body: unknown): void {
  const response = new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
}

function mockFetchNoContentOnce(): void {
  const response = new Response(null, { status: 204 });
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("login", () => {
  it("returns the username on success", async () => {
    mockFetchOnce(200, { username: "furkan" });
    const result = await login({ username: "furkan", password: "s3cret-password" });
    expect(result).toEqual({ ok: true, data: { username: "furkan" } });
  });

  it("returns the error on invalid credentials", async () => {
    mockFetchOnce(401, { error: { code: "invalid_credentials", message: "username or password is incorrect" } });
    const result = await login({ username: "furkan", password: "wrong" });
    expect(result).toEqual({
      ok: false,
      status: 401,
      error: { code: "invalid_credentials", message: "username or password is incorrect" },
    });
  });
});

describe("whoami", () => {
  it("returns the username when a session is active", async () => {
    mockFetchOnce(200, { username: "furkan" });
    const result = await whoami();
    expect(result).toEqual({ ok: true, data: { username: "furkan" } });
  });

  it("returns unauthorized when there is no session", async () => {
    mockFetchOnce(401, { error: { code: "unauthorized", message: "not logged in" } });
    const result = await whoami();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("unauthorized");
    }
  });
});

describe("logout", () => {
  it("resolves ok on 204", async () => {
    mockFetchNoContentOnce();
    const result = await logout();
    expect(result).toEqual({ ok: true, data: undefined });
  });
});

describe("changePassword", () => {
  it("resolves ok on 204", async () => {
    mockFetchNoContentOnce();
    const result = await changePassword({ current_password: "old", new_password: "new-password-123" });
    expect(result).toEqual({ ok: true, data: undefined });
  });

  it("returns the weak_password error", async () => {
    mockFetchOnce(400, { error: { code: "weak_password", message: "new password must be at least 8 characters" } });
    const result = await changePassword({ current_password: "old", new_password: "short" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("weak_password");
    }
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm test -- src/api/client.test.ts
```

Expected: FAIL — `./client` does not exist yet.

- [ ] **Step 3: Implement the API client**

`admin-ui/src/api/client.ts`:

```ts
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  username: string;
}

export interface WhoamiResponse {
  username: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

export interface ApiError {
  code: string;
  message: string;
}

export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: ApiError; status: number };

function isErrorBody(value: unknown): value is { error: ApiError } {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const candidate = value as { error?: unknown };
  if (typeof candidate.error !== "object" || candidate.error === null) {
    return false;
  }
  const err = candidate.error as { code?: unknown; message?: unknown };
  return typeof err.code === "string" && typeof err.message === "string";
}

async function parseJSON(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

function errorResult(parsed: unknown, status: number, statusText: string): ApiResult<never> {
  if (isErrorBody(parsed)) {
    return { ok: false, error: parsed.error, status };
  }
  return { ok: false, error: { code: "unknown_error", message: statusText }, status };
}

async function requestJSON<T>(path: string, init?: RequestInit): Promise<ApiResult<T>> {
  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  const parsed = await parseJSON(response);
  if (!response.ok) {
    return errorResult(parsed, response.status, response.statusText);
  }
  return { ok: true, data: parsed as T };
}

async function requestVoid(path: string, init?: RequestInit): Promise<ApiResult<void>> {
  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (response.status === 204) {
    return { ok: true, data: undefined };
  }
  const parsed = await parseJSON(response);
  return errorResult(parsed, response.status, response.statusText);
}

export function login(body: LoginRequest): Promise<ApiResult<LoginResponse>> {
  return requestJSON<LoginResponse>("/api/admin/login", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function logout(): Promise<ApiResult<void>> {
  return requestVoid("/api/admin/logout", { method: "POST" });
}

export function whoami(): Promise<ApiResult<WhoamiResponse>> {
  return requestJSON<WhoamiResponse>("/api/admin/whoami", { method: "GET" });
}

export function changePassword(body: ChangePasswordRequest): Promise<ApiResult<void>> {
  return requestVoid("/api/admin/change-password", {
    method: "POST",
    body: JSON.stringify(body),
  });
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/api/client.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/api
git commit -m "feat: add typed api client for admin auth endpoints"
```

---

### Task 4: Auth context and `useAuth` hook

**Files:**
- Create: `admin-ui/src/auth/AuthContext.tsx`
- Create: `admin-ui/src/auth/AuthContext.test.tsx`

**Interfaces:**
- Consumes: `login`, `logout`, `whoami`, `changePassword`, `ApiError` (Task 3)
- Produces: `AuthProvider`, `useAuth(): AuthState`, `AuthState{status, username, login, logout, changePassword}`

- [ ] **Step 1: Write the failing tests**

`admin-ui/src/auth/AuthContext.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { AuthProvider, useAuth } from "./AuthContext";
import * as client from "../api/client";

vi.mock("../api/client");

function Probe() {
  const { status, username } = useAuth();
  return (
    <div>
      <p data-testid="status">{status}</p>
      <p data-testid="username">{username ?? "none"}</p>
    </div>
  );
}

afterEach(() => {
  vi.resetAllMocks();
});

describe("AuthProvider", () => {
  it("resolves to anonymous when whoami fails", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });

    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    );

    expect(screen.getByTestId("status").textContent).toBe("loading");
    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("anonymous"));
    expect(screen.getByTestId("username").textContent).toBe("none");
  });

  it("resolves to authenticated when whoami succeeds", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    render(
      <AuthProvider>
        <Probe />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("authenticated"));
    expect(screen.getByTestId("username").textContent).toBe("furkan");
  });
});

function LoginProbe() {
  const { status, username, login } = useAuth();
  return (
    <div>
      <p data-testid="status">{status}</p>
      <p data-testid="username">{username ?? "none"}</p>
      <button onClick={() => void login("furkan", "s3cret-password")}>log in</button>
    </div>
  );
}

describe("useAuth login/logout", () => {
  it("updates status and username after a successful login", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });
    vi.mocked(client.login).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    render(
      <AuthProvider>
        <LoginProbe />
      </AuthProvider>,
    );

    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("anonymous"));
    screen.getByText("log in").click();

    await waitFor(() => expect(screen.getByTestId("status").textContent).toBe("authenticated"));
    expect(screen.getByTestId("username").textContent).toBe("furkan");
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm test -- src/auth/AuthContext.test.tsx
```

Expected: FAIL — `./AuthContext` does not exist yet.

- [ ] **Step 3: Implement the auth context**

`admin-ui/src/auth/AuthContext.tsx`:

```tsx
import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import {
  changePassword as apiChangePassword,
  login as apiLogin,
  logout as apiLogout,
  whoami as apiWhoami,
  type ApiError,
} from "../api/client";

export type AuthStatus = "loading" | "authenticated" | "anonymous";

export interface AuthState {
  status: AuthStatus;
  username: string | null;
  login: (username: string, password: string) => Promise<ApiError | null>;
  logout: () => Promise<void>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<ApiError | null>;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [username, setUsername] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void apiWhoami().then((result) => {
      if (cancelled) {
        return;
      }
      if (result.ok) {
        setUsername(result.data.username);
        setStatus("authenticated");
      } else {
        setUsername(null);
        setStatus("anonymous");
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (usernameInput: string, password: string): Promise<ApiError | null> => {
    const result = await apiLogin({ username: usernameInput, password });
    if (!result.ok) {
      return result.error;
    }
    setUsername(result.data.username);
    setStatus("authenticated");
    return null;
  }, []);

  const logout = useCallback(async (): Promise<void> => {
    await apiLogout();
    setUsername(null);
    setStatus("anonymous");
  }, []);

  const changePassword = useCallback(
    async (currentPassword: string, newPassword: string): Promise<ApiError | null> => {
      const result = await apiChangePassword({
        current_password: currentPassword,
        new_password: newPassword,
      });
      if (!result.ok) {
        return result.error;
      }
      return null;
    },
    [],
  );

  return (
    <AuthContext.Provider value={{ status, username, login, logout, changePassword }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/auth/AuthContext.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/auth
git commit -m "feat: add auth context resolving session state via whoami"
```

---

### Task 5: Login page

**Files:**
- Create: `admin-ui/src/pages/LoginPage.tsx`
- Create: `admin-ui/src/pages/LoginPage.test.tsx`

**Interfaces:**
- Consumes: `useAuth` (Task 4)
- Produces: `LoginPage` (default export component)

- [ ] **Step 1: Write the failing tests**

`admin-ui/src/pages/LoginPage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import LoginPage from "./LoginPage";
import * as authModule from "../auth/AuthContext";

vi.mock("../auth/AuthContext");

function renderLoginPage() {
  render(
    <MemoryRouter initialEntries={["/login"]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<p>Dashboard Home</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.resetAllMocks();
});

describe("LoginPage", () => {
  it("submits the entered credentials and navigates on success", async () => {
    const login = vi.fn().mockResolvedValue(null);
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "anonymous",
      username: null,
      login,
      logout: vi.fn(),
      changePassword: vi.fn(),
    });

    renderLoginPage();

    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "furkan" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "s3cret-password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => expect(login).toHaveBeenCalledWith("furkan", "s3cret-password"));
    await waitFor(() => expect(screen.getByText("Dashboard Home")).toBeInTheDocument());
  });

  it("shows the error message on failed login", async () => {
    const login = vi.fn().mockResolvedValue({ code: "invalid_credentials", message: "username or password is incorrect" });
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "anonymous",
      username: null,
      login,
      logout: vi.fn(),
      changePassword: vi.fn(),
    });

    renderLoginPage();

    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "furkan" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "wrong" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("username or password is incorrect"),
    );
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm test -- src/pages/LoginPage.test.tsx
```

Expected: FAIL — `./LoginPage` does not exist yet.

- [ ] **Step 3: Implement the login page**

`admin-ui/src/pages/LoginPage.tsx`:

```tsx
import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    const apiError = await login(username, password);
    setSubmitting(false);
    if (apiError) {
      setError(apiError.message);
      return;
    }
    navigate("/", { replace: true });
  }

  return (
    <main className="login-page">
      <h1>Pixabros Admin</h1>
      <form onSubmit={handleSubmit}>
        <label htmlFor="username">Username</label>
        <input
          id="username"
          name="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
          autoComplete="username"
          required
        />
        <label htmlFor="password">Password</label>
        <input
          id="password"
          name="password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete="current-password"
          required
        />
        {error ? <p role="alert">{error}</p> : null}
        <button type="submit" disabled={submitting}>
          {submitting ? "Signing in…" : "Sign in"}
        </button>
      </form>
    </main>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/pages/LoginPage.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/pages/LoginPage.tsx admin-ui/src/pages/LoginPage.test.tsx
git commit -m "feat: add login page"
```

---

### Task 6: Change-password page

**Files:**
- Create: `admin-ui/src/pages/ChangePasswordPage.tsx`
- Create: `admin-ui/src/pages/ChangePasswordPage.test.tsx`

**Interfaces:**
- Consumes: `useAuth().changePassword` (Task 4)
- Produces: `ChangePasswordPage` (default export component)

- [ ] **Step 1: Write the failing tests**

`admin-ui/src/pages/ChangePasswordPage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import ChangePasswordPage from "./ChangePasswordPage";
import * as authModule from "../auth/AuthContext";

vi.mock("../auth/AuthContext");

afterEach(() => {
  vi.resetAllMocks();
});

describe("ChangePasswordPage", () => {
  it("shows a success message and clears the fields after a successful change", async () => {
    const changePassword = vi.fn().mockResolvedValue(null);
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "authenticated",
      username: "furkan",
      login: vi.fn(),
      logout: vi.fn(),
      changePassword,
    });

    render(<ChangePasswordPage />);

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "old-password-1" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "new-password-123" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(changePassword).toHaveBeenCalledWith("old-password-1", "new-password-123"),
    );
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Password updated."));
    expect(screen.getByLabelText("Current password")).toHaveValue("");
    expect(screen.getByLabelText("New password")).toHaveValue("");
  });

  it("shows the weak_password error", async () => {
    const changePassword = vi
      .fn()
      .mockResolvedValue({ code: "weak_password", message: "new password must be at least 8 characters" });
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "authenticated",
      username: "furkan",
      login: vi.fn(),
      logout: vi.fn(),
      changePassword,
    });

    render(<ChangePasswordPage />);

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "old-password-1" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "short" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("new password must be at least 8 characters"),
    );
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm test -- src/pages/ChangePasswordPage.test.tsx
```

Expected: FAIL — `./ChangePasswordPage` does not exist yet.

- [ ] **Step 3: Implement the change-password page**

`admin-ui/src/pages/ChangePasswordPage.tsx`:

```tsx
import { useState, type FormEvent } from "react";
import { useAuth } from "../auth/AuthContext";

export default function ChangePasswordPage() {
  const { changePassword } = useAuth();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    setSuccess(false);
    const apiError = await changePassword(currentPassword, newPassword);
    setSubmitting(false);
    if (apiError) {
      setError(apiError.message);
      return;
    }
    setSuccess(true);
    setCurrentPassword("");
    setNewPassword("");
  }

  return (
    <section>
      <h1>Change password</h1>
      <form onSubmit={handleSubmit}>
        <label htmlFor="current-password">Current password</label>
        <input
          id="current-password"
          type="password"
          value={currentPassword}
          onChange={(event) => setCurrentPassword(event.target.value)}
          autoComplete="current-password"
          required
        />
        <label htmlFor="new-password">New password</label>
        <input
          id="new-password"
          type="password"
          value={newPassword}
          onChange={(event) => setNewPassword(event.target.value)}
          autoComplete="new-password"
          required
        />
        {error ? <p role="alert">{error}</p> : null}
        {success ? <p role="status">Password updated.</p> : null}
        <button type="submit" disabled={submitting}>
          {submitting ? "Saving…" : "Save"}
        </button>
      </form>
    </section>
  );
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/pages/ChangePasswordPage.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/pages/ChangePasswordPage.tsx admin-ui/src/pages/ChangePasswordPage.test.tsx
git commit -m "feat: add change-password page"
```

---

### Task 7: Routing, protected-route guard, shell, and dashboard placeholder

**Files:**
- Create: `admin-ui/src/components/ProtectedRoute.tsx`
- Create: `admin-ui/src/components/ProtectedRoute.test.tsx`
- Create: `admin-ui/src/pages/Shell.tsx`
- Create: `admin-ui/src/pages/Shell.test.tsx`
- Create: `admin-ui/src/pages/DashboardPage.tsx`
- Modify: `admin-ui/src/App.tsx`
- Modify: `admin-ui/src/App.test.tsx`

**Interfaces:**
- Consumes: `useAuth` (Task 4), `LoginPage` (Task 5), `ChangePasswordPage` (Task 6)
- Produces: `ProtectedRoute`, `Shell`, `DashboardPage`, the real `App` route tree (basename `/I-am-a-pixabro`)

- [ ] **Step 1: Write the failing test for `ProtectedRoute`**

`admin-ui/src/components/ProtectedRoute.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./ProtectedRoute";
import * as authModule from "../auth/AuthContext";

vi.mock("../auth/AuthContext");

function renderGuarded() {
  render(
    <MemoryRouter initialEntries={["/"]}>
      <Routes>
        <Route path="/login" element={<p>Login Screen</p>} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <p>Protected Content</p>
            </ProtectedRoute>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

afterEach(() => {
  vi.resetAllMocks();
});

describe("ProtectedRoute", () => {
  it("shows a loading indicator while status is loading", () => {
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "loading",
      username: null,
      login: vi.fn(),
      logout: vi.fn(),
      changePassword: vi.fn(),
    });
    renderGuarded();
    expect(screen.getByRole("status")).toHaveTextContent("Checking session…");
  });

  it("redirects to /login when anonymous", () => {
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "anonymous",
      username: null,
      login: vi.fn(),
      logout: vi.fn(),
      changePassword: vi.fn(),
    });
    renderGuarded();
    expect(screen.getByText("Login Screen")).toBeInTheDocument();
  });

  it("renders children when authenticated", () => {
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "authenticated",
      username: "furkan",
      login: vi.fn(),
      logout: vi.fn(),
      changePassword: vi.fn(),
    });
    renderGuarded();
    expect(screen.getByText("Protected Content")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm test -- src/components/ProtectedRoute.test.tsx
```

Expected: FAIL — `./ProtectedRoute` does not exist yet.

- [ ] **Step 3: Implement `ProtectedRoute`**

`admin-ui/src/components/ProtectedRoute.tsx`:

```tsx
import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { status } = useAuth();

  if (status === "loading") {
    return <p role="status">Checking session…</p>;
  }
  if (status === "anonymous") {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
npm test -- src/components/ProtectedRoute.test.tsx
```

Expected: PASS

- [ ] **Step 5: Write the failing test for `Shell`**

`admin-ui/src/pages/Shell.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import Shell from "./Shell";
import * as authModule from "../auth/AuthContext";

vi.mock("../auth/AuthContext");

afterEach(() => {
  vi.resetAllMocks();
});

describe("Shell", () => {
  it("shows the signed-in username and a logout control", () => {
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "authenticated",
      username: "furkan",
      login: vi.fn(),
      logout: vi.fn(),
      changePassword: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route path="/" element={<Shell />}>
            <Route index element={<p>Dashboard Body</p>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText("Signed in as furkan")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Log out" })).toBeInTheDocument();
    expect(screen.getByText("Dashboard Body")).toBeInTheDocument();
  });

  it("calls logout when the button is clicked", () => {
    const logout = vi.fn();
    vi.mocked(authModule.useAuth).mockReturnValue({
      status: "authenticated",
      username: "furkan",
      login: vi.fn(),
      logout,
      changePassword: vi.fn(),
    });

    render(
      <MemoryRouter initialEntries={["/"]}>
        <Routes>
          <Route path="/" element={<Shell />}>
            <Route index element={<p>Dashboard Body</p>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );

    screen.getByRole("button", { name: "Log out" }).click();
    expect(logout).toHaveBeenCalled();
  });
});
```

- [ ] **Step 6: Run test to verify it fails**

```bash
npm test -- src/pages/Shell.test.tsx
```

Expected: FAIL — `./Shell` does not exist yet.

- [ ] **Step 7: Implement `Shell` and `DashboardPage`**

`admin-ui/src/pages/Shell.tsx`:

```tsx
import { Link, Outlet } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export default function Shell() {
  const { username, logout } = useAuth();

  return (
    <div className="shell">
      <header>
        <span>Signed in as {username}</span>
        <nav>
          <Link to="/">Dashboard</Link>
          <Link to="/change-password">Change password</Link>
        </nav>
        <button type="button" onClick={() => void logout()}>
          Log out
        </button>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  );
}
```

`admin-ui/src/pages/DashboardPage.tsx`:

```tsx
export default function DashboardPage() {
  return (
    <section>
      <h1>Dashboard</h1>
      <p>Module screens (Games, Members, Devlog, Awards, Contact, Site Settings, Media) land here in later plans.</p>
    </section>
  );
}
```

- [ ] **Step 8: Run test to verify it passes**

```bash
npm test -- src/pages/Shell.test.tsx
```

Expected: PASS

- [ ] **Step 9: Rewrite `App.tsx` with the real route tree**

`admin-ui/src/App.tsx`:

```tsx
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import { ProtectedRoute } from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import Shell from "./pages/Shell";
import DashboardPage from "./pages/DashboardPage";
import ChangePasswordPage from "./pages/ChangePasswordPage";

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter basename="/I-am-a-pixabro">
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Shell />
              </ProtectedRoute>
            }
          >
            <Route index element={<DashboardPage />} />
            <Route path="change-password" element={<ChangePasswordPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
```

- [ ] **Step 10: Rewrite `App.test.tsx` to cover both auth states**

`App` renders a `BrowserRouter` with `basename="/I-am-a-pixabro"`, but jsdom's default test location is `/`, which does not start with that basename — react-router would fail to match any route. Point the test's location at the basename before each render and restore it afterward:

`admin-ui/src/App.test.tsx`:

```tsx
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import App from "./App";
import * as client from "./api/client";

vi.mock("./api/client");

beforeEach(() => {
  window.history.pushState({}, "", "/I-am-a-pixabro/");
});

afterEach(() => {
  vi.resetAllMocks();
  window.history.pushState({}, "", "/");
});

describe("App", () => {
  it("shows the login page when anonymous", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });

    render(<App />);

    await waitFor(() => expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument());
  });

  it("shows the dashboard when authenticated", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    render(<App />);

    await waitFor(() => expect(screen.getByText("Signed in as furkan")).toBeInTheDocument());
    expect(screen.getByRole("heading", { name: "Dashboard" })).toBeInTheDocument();
  });
});
```

- [ ] **Step 11: Run the full frontend test suite**

```bash
npm test
```

Expected: PASS for every test file.

- [ ] **Step 12: Commit**

```bash
git add admin-ui/src/App.tsx admin-ui/src/App.test.tsx admin-ui/src/components admin-ui/src/pages/Shell.tsx admin-ui/src/pages/Shell.test.tsx admin-ui/src/pages/DashboardPage.tsx
git commit -m "feat: add routing, protected-route guard, shell, and dashboard placeholder"
```

---

### Task 8: Service Worker for app-shell asset caching

**Files:**
- Create: `admin-ui/public/sw.js`
- Create: `admin-ui/src/registerServiceWorker.ts`
- Create: `admin-ui/src/registerServiceWorker.test.ts`
- Modify: `admin-ui/src/main.tsx`

**Interfaces:**
- Consumes: nothing from earlier tasks (standalone)
- Produces: `shouldRegisterServiceWorker(isProd, hasServiceWorkerSupport): boolean`, `registerServiceWorker(): void`

- [ ] **Step 1: Write the failing tests for the registration logic**

`admin-ui/src/registerServiceWorker.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { registerServiceWorker, shouldRegisterServiceWorker } from "./registerServiceWorker";

describe("shouldRegisterServiceWorker", () => {
  it("registers only in production with service worker support", () => {
    expect(shouldRegisterServiceWorker(true, true)).toBe(true);
  });

  it("does not register outside production", () => {
    expect(shouldRegisterServiceWorker(false, true)).toBe(false);
  });

  it("does not register without browser support", () => {
    expect(shouldRegisterServiceWorker(true, false)).toBe(false);
  });

  it("does not register in dev without support", () => {
    expect(shouldRegisterServiceWorker(false, false)).toBe(false);
  });
});

describe("registerServiceWorker", () => {
  it("does nothing in the jsdom test environment (no service worker support, not production)", () => {
    expect(() => registerServiceWorker()).not.toThrow();
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm test -- src/registerServiceWorker.test.ts
```

Expected: FAIL — `./registerServiceWorker` does not exist yet.

- [ ] **Step 3: Implement the registration module**

`admin-ui/src/registerServiceWorker.ts`:

```ts
export function shouldRegisterServiceWorker(isProd: boolean, hasServiceWorkerSupport: boolean): boolean {
  return isProd && hasServiceWorkerSupport;
}

export function registerServiceWorker(): void {
  if (!shouldRegisterServiceWorker(import.meta.env.PROD, "serviceWorker" in navigator)) {
    return;
  }
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/I-am-a-pixabro/sw.js").catch((error: unknown) => {
      console.error("service worker registration failed", error);
    });
  });
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/registerServiceWorker.test.ts
```

Expected: PASS

- [ ] **Step 5: Write the Service Worker source file**

`admin-ui/public/sw.js` (Vite copies `public/` verbatim to `build.outDir`, so this ships as `{AdminUIDir}/sw.js`, served at `/I-am-a-pixabro/sw.js`; its scope defaults to that path, so the browser never dispatches `fetch` events to it for anything outside `/I-am-a-pixabro/` — in particular never for `/api/*`, which lives at a sibling path):

```js
const CACHE_NAME = "pixabros-admin-shell-v1";
const APP_SHELL_SCOPE = "/I-am-a-pixabro/";

self.addEventListener("install", (event) => {
  self.skipWaiting();
  event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.add(APP_SHELL_SCOPE)));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);

  // Only ever cache GET requests for the admin app shell itself.
  if (event.request.method !== "GET" || !url.pathname.startsWith(APP_SHELL_SCOPE)) {
    return;
  }
  // Never intercept the API, even though it is outside our scope anyway —
  // kept explicit so this rule is auditable in one place.
  if (url.pathname.startsWith("/api/")) {
    return;
  }

  event.respondWith(
    caches.open(CACHE_NAME).then(async (cache) => {
      const cached = await cache.match(event.request);
      if (cached) {
        return cached;
      }
      const response = await fetch(event.request);
      if (response.ok) {
        cache.put(event.request, response.clone());
      }
      return response;
    }),
  );
});
```

- [ ] **Step 6: Wire registration into the entrypoint**

Modify `admin-ui/src/main.tsx`:

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { registerServiceWorker } from "./registerServiceWorker";
import "./index.css";

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("root element not found");
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
);

registerServiceWorker();
```

- [ ] **Step 7: Manual browser verification**

`jsdom` has no `ServiceWorkerGlobalScope`, so the SW's `fetch`-interception behavior cannot be unit tested — verify it manually once the shell is built and served (after Task 10's server is running):

```bash
open http://localhost:8080/I-am-a-pixabro/
```

In Chrome DevTools → Application → Service Workers: confirm `sw.js` is `activated and running` with scope `/I-am-a-pixabro/`. In the Network tab, reload and confirm hashed JS/CSS assets under `/I-am-a-pixabro/assets/` show `(ServiceWorker)` as their source on the second load, while any `/api/admin/*` request always shows a real network fetch, never `(ServiceWorker)`.

- [ ] **Step 8: Run the full frontend test suite**

```bash
npm test
```

Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add admin-ui/public/sw.js admin-ui/src/registerServiceWorker.ts admin-ui/src/registerServiceWorker.test.ts admin-ui/src/main.tsx
git commit -m "feat: add service worker for app-shell asset caching"
```

---

### Task 9: Build output wiring into the Go server's expected directory

**Files:**
- Create: `Makefile`
- Modify: `.gitignore` (repo root, from Plan A)

**Interfaces:**
- Consumes: `admin-ui`'s `npm run build` (Task 2's `vite.config.ts` `outDir` logic)
- Produces: `make admin-build` — builds the SPA straight into the exact directory `cmd/server/main.go` serves from (`{DataDir}/admin-dist`, default `./data/admin-dist`)

- [ ] **Step 1: Add a root Makefile target**

`Makefile`:

```makefile
.PHONY: admin-build
admin-build:
	npm --prefix admin-ui ci
	npm --prefix admin-ui run build
```

`ADMIN_UI_OUT_DIR` (read by `admin-ui/vite.config.ts`, Task 2 Step 6) lets deployments whose `PIXABROS_DATA_DIR` differs from the default `./data` point the build at the matching directory, e.g.:

```bash
ADMIN_UI_OUT_DIR=/srv/pixabros/data/admin-dist make admin-build
```

- [ ] **Step 2: Extend the root `.gitignore`**

Append to `.gitignore` (from Plan A Task 1 — it already ignores `/data/`, which covers the build *output* since it lands under `./data/admin-dist`; this adds the frontend's own build tooling artifacts):

```gitignore
/admin-ui/node_modules/
```

- [ ] **Step 3: Run the build and verify the output lands in the right place**

```bash
make admin-build
ls data/admin-dist
```

Expected: `data/admin-dist/index.html` and `data/admin-dist/assets/` (containing content-hashed `.js`/`.css` files) and `data/admin-dist/sw.js` all exist.

- [ ] **Step 4: Verify the base path is baked into the built HTML**

```bash
grep -o '/I-am-a-pixabro/assets/[^"]*\.js' data/admin-dist/index.html
```

Expected: at least one match — confirms Vite's `base: "/I-am-a-pixabro/"` (Task 2 Step 6) was applied to the production build's asset URLs.

- [ ] **Step 5: Commit**

```bash
git add Makefile .gitignore
git commit -m "feat: add make target building the admin ui into the server's expected directory"
```

---

### Task 10: End-to-end verification against the Go server

**Files:**
- None (verification only)

**Interfaces:**
- Consumes: everything from Tasks 1–9

- [ ] **Step 1: Build both halves of the system**

```bash
go build ./...
make admin-build
mkdir -p data/games data/rendered-store data/assets
```

- [ ] **Step 2: Seed an admin and start the server**

```bash
go run ./cmd/admincli create-admin -username furkan -password "a-strong-password-1"
go run ./cmd/server &
sleep 1
```

- [ ] **Step 3: Verify the anonymous flow**

```bash
curl -i http://localhost:8080/I-am-a-pixabro/
curl -i http://localhost:8080/api/admin/whoami
```

Expected: the first request returns `200` with the built SPA's `index.html` (containing a `/I-am-a-pixabro/assets/...js` script tag); the second returns `401` with `{"error":{"code":"unauthorized","message":"not logged in"}}`.

- [ ] **Step 4: Verify the authenticated flow**

```bash
curl -i -c /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"furkan","password":"a-strong-password-1"}'
curl -i -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/whoami
curl -i -b /tmp/pixabros-cookies.txt http://localhost:8080/I-am-a-pixabro/sw.js
```

Expected: login returns `200` with `Set-Cookie: pixabros_session=...; HttpOnly; Secure; SameSite=Strict` and body `{"username":"furkan"}`; the second `whoami` call returns `200` with the same body; the `sw.js` request returns `200` with the Service Worker source from Task 8.

- [ ] **Step 5: Verify the change-password and logout flow**

```bash
curl -i -b /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/change-password \
  -H 'Content-Type: application/json' \
  -d '{"current_password":"a-strong-password-1","new_password":"a-different-strong-password-2"}'
curl -i -b /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/logout
curl -i -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/whoami
```

Expected: change-password returns `204`; logout returns `204` and clears the cookie; the final `whoami` call (cookie now invalid) returns `401`.

- [ ] **Step 6: Open the SPA in a real browser and click through it**

```bash
open http://localhost:8080/I-am-a-pixabro/
```

Log in with `furkan` / `a-different-strong-password-2`, confirm the dashboard placeholder renders with "Signed in as furkan", navigate to "Change password", submit a new password, then click "Log out" and confirm you land back on the login screen. Complete Task 8 Step 7's DevTools Service Worker check here too.

- [ ] **Step 7: Stop the server and run the full test suites one last time**

```bash
kill %1
go test ./...
(cd admin-ui && npm test)
```

Expected: PASS for every Go package and every Vitest file.

- [ ] **Step 8: Final commit (only if any stray files remain from verification)**

```bash
git status
```

If everything from Tasks 1–9 is already committed, there is nothing left to commit here — this task is verification-only.

---

## Definition of Done

- `go build ./...` and `go test ./...` succeed, including the new `Whoami` handler and router tests.
- `cd admin-ui && npm test` passes with no skipped files; `npm run lint` reports zero errors (in particular, zero `@typescript-eslint/no-explicit-any` violations).
- `make admin-build` produces `data/admin-dist/index.html` + a content-hashed `assets/` bundle + `sw.js`, with `/I-am-a-pixabro/` baked into every asset URL.
- Against a running `cmd/server`: an anonymous `GET /api/admin/whoami` returns `401`; `POST /api/admin/login` with valid credentials sets the `pixabros_session` cookie and a subsequent `whoami` returns `200` with the username; `POST /api/admin/change-password` and `POST /api/admin/logout` behave per their Plan A contracts; the logged-out session's `whoami` call returns `401` again.
- In a real browser at `http://localhost:8080/I-am-a-pixabro/`: login → dashboard → change-password → logout all work end-to-end, and DevTools confirms the Service Worker is active with scope `/I-am-a-pixabro/`, serves hashed JS/CSS from cache on repeat loads, and never serves `/api/*` from cache.
