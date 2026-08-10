# Admin SPA Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the React + Vite admin SPA *shell*, styled per the approved design spec: project scaffold, login page, a minimal authenticated shell/dashboard, a change-password screen, logout, and a Service Worker for app-shell asset caching — served by Plan A/B's Go backend at `/I-am-a-pixabro/*`. Adds the one missing backend primitive (`GET /api/admin/whoami`) the SPA needs to restore auth state after a page refresh, since the session cookie is `HttpOnly` and unreadable from JS.

**Architecture:** A TypeScript React SPA (Vite-built), talking to the existing `/api/admin/*` JSON endpoints over `fetch` with `credentials: "include"`. Auth state lives in a TanStack Query `whoami` query plus login/logout/change-password mutations that update that query's cache directly. `react-router-dom` handles the (currently tiny) route tree so later per-module CRUD screens slot in without restructuring. A hand-written Service Worker caches only the built app-shell assets (cache-first) and never touches `/api/*`. Visuals follow `docs/superpowers/specs/2026-08-10-frontend-design-and-stack.md` exactly: dark theme, fixed color tokens, Inter font, Tailwind CSS v3, shadcn/ui + Radix components, Formik + Yup forms, sonner toasts.

**Architecture Decisions:**

1. **TypeScript, not plain JS.** The user's global CLAUDE.md rule bans `any` everywhere, including TS. Adopting TypeScript makes that rule enforceable by the compiler and linter instead of by discipline alone.
2. **Auth state via TanStack Query's `whoami` query, not a client-readable marker cookie.** Because `pixabros_session` is `HttpOnly`, no client-side check can tell the SPA "am I logged in?" after a refresh. Plan A's `internal/adminapi` already has the session-validating middleware (`RequireSession`) and everything `Whoami` needs (`AdminRepo.FindByID`, `AdminIDFromContext`) — adding a tiny `GET /api/admin/whoami` handler that reuses them is far simpler than any client-only alternative.
3. **`react-router-dom`, not conditional rendering.** A shell with 3 screens *could* be done with `if`/`else`, but every future per-module plan (Games, Members, Devlog, Awards, Contact, Site Settings, Media, regen-job status) needs its own URL for deep-linking and bookmarking inside the admin panel.
4. **Hand-written Service Worker, not `vite-plugin-pwa`.** The caching rule is simple and fixed for the life of this shell: cache-first for built JS/CSS/HTML under `/I-am-a-pixabro/`, network-only (never touched) for `/api/*`. A 20-line hand-written `public/sw.js` makes that rule auditable in one file.
5. **Vitest + React Testing Library, real assertions.** Every component is rendered and asserted against real DOM output — no snapshot placeholders. The Service Worker's *behavioral* logic (which requests to intercept) is extracted into a plain, unit-testable function; the SW file itself is verified manually in a browser, since `jsdom` has no `ServiceWorkerGlobalScope`.
6. **Tailwind CSS v3 (the user's explicit choice).** Utility-first styling matches the setup shadcn/ui's generated components expect out of the box, and v3's stable JIT compiler needs no build-step complexity beyond the PostCSS pipeline Vite already runs.
7. **Formik + Yup (the user's explicit choice).** A mature, widely used form/validation pair; Yup's schema mirrors the backend's own validation rules (e.g., the 8-character password minimum) so the user gets instant client-side feedback before a request ever fires.
8. **TanStack Query (the user's explicit choice).** Removes hand-rolled loading/error/cache state for `whoami` and the three mutations; `queryClient.setQueryData` after login/logout keeps the cache in sync without a redundant network round-trip.
9. **sonner (the user's explicit choice).** A lightweight toast library. It's used *alongside*, never instead of, inline error/status text — so tests can assert on visible DOM without touching a toast portal, while real users still get the transient toast affordance.
10. **shadcn/ui + Radix, with `cssVariables: false`.** This project has exactly one fixed dark theme with no runtime switching, so components reference the named Tailwind tokens directly instead of an indirection layer of CSS variables. Components are copied into the repo (shadcn's standard workflow), not consumed as an opaque npm package, so the exact visual identity from the design spec stays fully under our control.
11. **`classnames` (the user's explicit choice) + `tailwind-merge`.** `classnames` is the class-joining utility the user asked for; `tailwind-merge` is kept alongside it because it solves a distinct problem — deduplicating/overriding conflicting Tailwind utility classes when a consumer passes an extra `className` prop — that `classnames` doesn't address. Both are combined into one small `cn()` helper so every component calls one function, not two libraries directly.

**Tech Stack:** Go 1.22+ (whoami addition only), Vite 5, React 18, TypeScript 5 (strict), `react-router-dom` 6, Tailwind CSS v3, `@fontsource/inter`, Formik + Yup, TanStack Query (React Query) v5, sonner, `classnames` + `tailwind-merge`, shadcn/ui + Radix primitives (`@radix-ui/react-slot`, `@radix-ui/react-label`, `class-variance-authority`), Vitest 2 + `@testing-library/react` + `@testing-library/jest-dom`, ESLint 8 + `@typescript-eslint` (with `no-explicit-any` as an error).

**Depends on:** `docs/superpowers/plans/2026-08-10-backend-core-data-model.md` (Plan A), `docs/superpowers/plans/2026-08-10-content-rendering-pipelines.md` (Plan B), and `docs/superpowers/specs/2026-08-10-frontend-design-and-stack.md` (the design spec this plan implements). This plan assumes Plan A/B already exist and their tests pass, and it modifies the **post-Plan-B** state of `internal/httpserver/router.go` (`Dependencies{Admins, Sessions, Store, Files, AdminUIDir, PlayDir, AssetsDir}`) and relies on, but does not modify, `cmd/server/main.go`.

## Global Constraints

- Never use TypeScript's `any` — use `unknown` + type guards, or precise generics, everywhere (user's global CLAUDE.md rule; enforced here via `@typescript-eslint/no-explicit-any: "error"`). Never use Go's `any` alias either (unchanged from Plan A/B).
- Session cookie contract is fixed by Plan A: name `pixabros_session`, `HttpOnly, Secure, SameSite=Strict`. The SPA never reads it directly.
- Every API response follows `{"error": {"code": "...", "message": "..."}}` on failure; the client's type guards must not assume any other shape.
- Colors, typography, and every other visual token come from `docs/superpowers/specs/2026-08-10-frontend-design-and-stack.md` exactly — no ad-hoc colors. Background `#0F1115`, surface `#171A21`, text `#F1F1F3`, muted text `#9AA0AC`, border `#2A2E37`, accent `#E879F9` (hover `#C026D3`), success `#34D399`, error `#F87171`, warning `#FBBF24`. Font: Inter everywhere in the admin panel (the retro pixel font from the design spec is Play-page-only, on the public site, never in the admin panel).
- This is a single, fixed dark theme — no runtime theme switching, no light-mode variant, no `cssVariables` indirection in shadcn's config.
- Tailwind CSS **v3** specifically (not v4) — this is what the user asked for.
- The SPA is mounted at `/I-am-a-pixabro/*` only — Vite's `base` and `react-router-dom`'s `basename` must both reflect that exact path.
- The Service Worker's scope is `/I-am-a-pixabro/` (its own registration path); it must never intercept `/api/*` requests.
- Per-module CRUD screens (Games, Members, Devlog, Awards, Contact inbox, Homepage/Site Settings, Media library, regen-job status) are **out of scope** — their backend APIs don't exist yet and land in later per-module plans.
- Git commits in this repo: self-committed, one-sentence semantic messages, no co-author trailer.

## Scope

This plan (Plan C of three) builds only the admin SPA shell: scaffold, login, minimal authenticated dashboard, change-password, logout, Service Worker, and the one backend addition (`whoami`) the shell needs — now built with the confirmed design language and library stack instead of unstyled placeholders. Out of scope, deferred to future per-module plans:
- Any Games/Members/Devlog/Awards/Contact/Site-Settings/Media/regen-job CRUD screens or their backend REST endpoints.
- Contact-form honeypot/rate-limiting (that's public-site scope, not admin).
- Public-site MPA rendering and its plain-CSS styling (separate, covered by Plan B + the design spec's public-site section).

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
  tailwind.config.js
  postcss.config.js
  components.json
  public/
    sw.js
  src/
    vite-env.d.ts
    main.tsx
    App.tsx
    App.test.tsx
    index.css
    setupTests.ts
    testUtils.tsx
    registerServiceWorker.ts
    registerServiceWorker.test.ts
    lib/
      utils.ts
    api/
      client.ts
      client.test.ts
    auth/
      queries.ts
      queries.test.tsx
    components/
      ProtectedRoute.tsx
      ProtectedRoute.test.tsx
      ui/
        button.tsx
        input.tsx
        label.tsx
        ui.test.tsx
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

Each `admin-ui/src/*` directory owns one concern (typed API access, auth query/mutation hooks, shadcn UI primitives, route guarding, screens). `internal/adminapi` and `internal/httpserver` get small, additive modifications — no existing Plan A/B behavior changes.

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

### Task 2: Vite + React + TypeScript scaffold (with the full dependency set)

This scaffold's `package.json` declares **every** dependency used across the whole plan up front — Tailwind, shadcn's building blocks, Formik/Yup, TanStack Query, sonner, classnames — so later tasks never need a second `npm install`; they only add files and config.

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
- Create: `admin-ui/src/main.tsx` (placeholder; rewritten in Task 6, then Task 10)

**Interfaces:**
- Consumes: nothing from earlier tasks
- Produces: a buildable, testable Vite project — `npm run build`, `npm test`, `npm run lint` all work

- [ ] **Step 1: Create `package.json` with the full dependency set**

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
    "@fontsource/inter": "^5.1.0",
    "@radix-ui/react-label": "^2.1.0",
    "@radix-ui/react-slot": "^1.1.0",
    "@tanstack/react-query": "^5.59.0",
    "class-variance-authority": "^0.7.0",
    "classnames": "^2.5.1",
    "formik": "^2.4.6",
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.26.2",
    "sonner": "^1.5.0",
    "tailwind-merge": "^2.5.2",
    "yup": "^1.4.0"
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
    "autoprefixer": "^10.4.20",
    "eslint": "^8.57.1",
    "eslint-plugin-react-hooks": "^4.6.2",
    "eslint-plugin-react-refresh": "^0.4.12",
    "jsdom": "^25.0.1",
    "postcss": "^8.4.47",
    "tailwindcss": "^3.4.13",
    "typescript": "^5.6.2",
    "vite": "^5.4.9",
    "vitest": "^2.1.2"
  }
}
```

Every dependency any later task imports is already listed here: `tailwindcss`/`postcss`/`autoprefixer` (Task 3), `@fontsource/inter` (Task 3), `@radix-ui/react-label`/`@radix-ui/react-slot`/`class-variance-authority`/`classnames`/`tailwind-merge` (Task 5), `@tanstack/react-query`/`sonner` (Task 6), `formik`/`yup` (Tasks 7–8), `react-router-dom` (Task 9). No later task needs a second `npm install` for a *new* package — only `npm install` re-runs (idempotent, no-op if lockfile unchanged) as part of `make admin-build` (Task 11).

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

- [ ] **Step 4: Add TypeScript config, with the `@/*` alias shadcn's generated components expect**

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
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] },
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

- [ ] **Step 6: Add Vite config (base path, output dir, `@/*` alias, Vitest settings)**

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
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
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

- [ ] **Step 8: Add `vite-env.d.ts` and a minimal `index.css`**

`admin-ui/src/vite-env.d.ts`:

```ts
/// <reference types="vite/client" />
```

`admin-ui/src/index.css` (placeholder — fully replaced in Task 3 with Tailwind directives and the design tokens):

```css
body {
  margin: 0;
  font-family: system-ui, sans-serif;
}
```

- [ ] **Step 9: Add the Vitest setup file**

`admin-ui/src/setupTests.ts`:

```ts
import "@testing-library/jest-dom/vitest";
```

- [ ] **Step 10: Write the failing smoke test**

`admin-ui/src/App.test.tsx` (placeholder — fully replaced in Task 9 to cover real routing):

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

`admin-ui/src/App.tsx` (placeholder — fully replaced in Task 9 with real routing; kept minimal here to prove the toolchain end-to-end):

```tsx
export default function App() {
  return <div>Pixabros Admin</div>;
}
```

`admin-ui/src/main.tsx` (placeholder — modified in Task 3 for fonts, Task 6 for TanStack Query + sonner, Task 10 for Service Worker registration):

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
git commit -m "feat: scaffold vite react typescript admin ui project with full dependency set"
```

---

### Task 3: Tailwind CSS v3 setup — design tokens + Inter font

**Files:**
- Create: `admin-ui/tailwind.config.js`
- Create: `admin-ui/postcss.config.js`
- Modify: `admin-ui/src/index.css`
- Modify: `admin-ui/src/main.tsx`

**Interfaces:**
- Consumes: nothing from earlier tasks besides the scaffold
- Produces: Tailwind utility classes `bg-background`, `bg-surface`, `text-text`, `text-muted`, `border-border`, `bg-accent` / `hover:bg-accent-dark` / `text-accent-foreground`, `bg-success`/`text-success`, `bg-error`/`text-error`, `bg-warning`/`text-warning`, and the `font-sans` family set to Inter

- [ ] **Step 1: Write the CSS that should fail to compile into real utility classes without a Tailwind config**

Modify `admin-ui/src/index.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    color-scheme: dark;
  }

  body {
    @apply bg-background text-text font-sans antialiased;
    margin: 0;
  }
}
```

Modify `admin-ui/src/main.tsx` (add the self-hosted Inter font imports; everything else unchanged from Task 2):

```tsx
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/inter/600.css";
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

- [ ] **Step 2: Build and confirm the tokens are NOT yet compiled (no Tailwind/PostCSS config exists)**

```bash
cd admin-ui
npm run build
grep -ri "0f1115" data/admin-dist/assets/*.css 2>/dev/null || grep -ri "0f1115" ../data/admin-dist/assets/*.css
```

Expected: FAIL to find the token — either the build errors on the unrecognized `@apply`/`@tailwind` at-rules, or it succeeds but emits them uninterpreted, so no compiled `#0f1115` rule exists in the output CSS yet.

- [ ] **Step 3: Add the Tailwind config with the exact design-spec tokens**

`admin-ui/tailwind.config.js`:

```js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        background: "#0F1115",
        surface: "#171A21",
        text: "#F1F1F3",
        muted: "#9AA0AC",
        border: "#2A2E37",
        accent: {
          DEFAULT: "#E879F9",
          dark: "#C026D3",
          foreground: "#0F1115",
        },
        success: "#34D399",
        error: "#F87171",
        warning: "#FBBF24",
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
      },
    },
  },
  plugins: [],
};
```

- [ ] **Step 4: Add the PostCSS config wiring Tailwind + Autoprefixer**

`admin-ui/postcss.config.js`:

```js
module.exports = {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
};
```

- [ ] **Step 5: Rebuild and confirm the tokens now compile into real CSS**

```bash
npm run build
grep -ri "0f1115" ../data/admin-dist/assets/*.css
grep -ri "e879f9" ../data/admin-dist/assets/*.css
```

Expected: PASS — both hex tokens appear in the compiled CSS asset (Tailwind lowercases hex output), because `index.css`'s `@layer base` block applies `bg-background`/`text-text` directly on `body`, guaranteeing those utilities are never purged even before other components use them.

- [ ] **Step 6: Run the frontend test suite (unaffected by CSS, should still pass)**

```bash
npm test
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add admin-ui/tailwind.config.js admin-ui/postcss.config.js admin-ui/src/index.css admin-ui/src/main.tsx
git commit -m "feat: add tailwind css v3 with design-spec color tokens and inter font"
```

---

### Task 4: Typed API client

Unchanged in shape from the pre-stack-decision version of this plan — this file is a plain `fetch` wrapper with no UI framework dependency, so the stack change doesn't affect it. It's the seam every later TanStack Query hook is built on.

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

### Task 5: shadcn/ui setup — `cn()` helper, `Button`, `Input`, `Label`

**Files:**
- Create: `admin-ui/components.json`
- Create: `admin-ui/src/lib/utils.ts`
- Create: `admin-ui/src/components/ui/button.tsx`
- Create: `admin-ui/src/components/ui/input.tsx`
- Create: `admin-ui/src/components/ui/label.tsx`
- Create: `admin-ui/src/components/ui/ui.test.tsx`

**Interfaces:**
- Consumes: Tailwind tokens from Task 3 (`accent`, `accent-dark`, `accent-foreground`, `surface`, `border`, `text`, `muted`, `background`)
- Produces: `cn(...inputs: ClassValue[]): string`; `Button`, `Input`, `Label` components under `src/components/ui/`

- [ ] **Step 1: Add the shadcn config file**

`admin-ui/components.json` (recorded as if generated by `npx shadcn init`, with `cssVariables: false` because this project has exactly one fixed dark theme — no runtime theme switching — so components reference the named Tailwind tokens directly instead of an indirection layer of CSS variables):

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": false,
  "tsx": true,
  "tailwind": {
    "config": "tailwind.config.js",
    "css": "src/index.css",
    "baseColor": "slate",
    "cssVariables": false,
    "prefix": ""
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils",
    "ui": "@/components/ui"
  }
}
```

- [ ] **Step 2: Write the failing tests for `Button`, `Input`, `Label`**

`admin-ui/src/components/ui/ui.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { Button } from "./button";
import { Input } from "./input";
import { Label } from "./label";

describe("Button", () => {
  it("renders as a button with the given label and responds to clicks", () => {
    let clicked = false;
    render(<Button onClick={() => (clicked = true)}>Sign in</Button>);
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(clicked).toBe(true);
  });

  it("applies the accent background class by default", () => {
    render(<Button>Save</Button>);
    expect(screen.getByRole("button", { name: "Save" }).className).toContain("bg-accent");
  });
});

describe("Input and Label", () => {
  it("associates a label with its input via htmlFor/id", () => {
    render(
      <div>
        <Label htmlFor="email">Email</Label>
        <Input id="email" onChange={() => undefined} />
      </div>,
    );
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("reflects typed input value", () => {
    render(<Input aria-label="Username" onChange={() => undefined} />);
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "furkan" } });
    expect(screen.getByLabelText("Username")).toHaveValue("furkan");
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
npm test -- src/components/ui/ui.test.tsx
```

Expected: FAIL — `./button`, `./input`, `./label` do not exist yet.

- [ ] **Step 4: Add the `cn()` helper**

`admin-ui/src/lib/utils.ts` — standardizes on `classnames` (the library the user explicitly named) instead of `clsx` for joining class strings, but keeps `tailwind-merge` because it solves a different problem (deduplicating/overriding conflicting Tailwind utility classes when a consumer passes an extra `className` prop) that `classnames` doesn't address:

```ts
import classNames from "classnames";
import { twMerge } from "tailwind-merge";

export type ClassValue = string | number | boolean | null | undefined | Record<string, boolean | null | undefined>;

export function cn(...inputs: ClassValue[]): string {
  return twMerge(classNames(...inputs));
}
```

- [ ] **Step 5: Implement `Button`**

`admin-ui/src/components/ui/button.tsx`:

```tsx
import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-accent text-accent-foreground hover:bg-accent-dark",
        outline: "border border-border bg-transparent text-text hover:bg-surface",
        ghost: "bg-transparent text-text hover:bg-surface",
        destructive: "bg-error text-background hover:bg-error/90",
      },
      size: {
        default: "h-10 px-4 py-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return <Comp className={cn(buttonVariants({ variant, size }), className)} ref={ref} {...props} />;
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
```

- [ ] **Step 6: Implement `Input`**

`admin-ui/src/components/ui/input.tsx`:

```tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(({ className, type, ...props }, ref) => {
  return (
    <input
      type={type}
      className={cn(
        "flex h-10 w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      ref={ref}
      {...props}
    />
  );
});
Input.displayName = "Input";

export { Input };
```

- [ ] **Step 7: Implement `Label`**

`admin-ui/src/components/ui/label.tsx`:

```tsx
import * as React from "react";
import * as LabelPrimitive from "@radix-ui/react-label";
import { cn } from "@/lib/utils";

const Label = React.forwardRef<
  React.ElementRef<typeof LabelPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof LabelPrimitive.Root>
>(({ className, ...props }, ref) => (
  <LabelPrimitive.Root ref={ref} className={cn("text-sm font-medium leading-none text-text", className)} {...props} />
));
Label.displayName = LabelPrimitive.Root.displayName;

export { Label };
```

- [ ] **Step 8: Run tests to verify they pass**

```bash
npm test -- src/components/ui/ui.test.tsx
```

Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add admin-ui/components.json admin-ui/src/lib admin-ui/src/components/ui
git commit -m "feat: add shadcn/ui config and button, input, label components"
```

---

### Task 6: TanStack Query provider + auth query/mutation hooks

**Files:**
- Create: `admin-ui/src/testUtils.tsx`
- Create: `admin-ui/src/auth/queries.ts`
- Create: `admin-ui/src/auth/queries.test.tsx`
- Modify: `admin-ui/src/main.tsx`

**Interfaces:**
- Consumes: `login`, `logout`, `whoami`, `changePassword`, `ApiResult`, `WhoamiResponse` (Task 4)
- Produces: `createTestQueryClient()`, `renderWithQueryClient(ui, options?)`, `whoamiQueryKey`, `useWhoamiStatus(): { status: "loading" | "authenticated" | "anonymous"; username: string | null }`, `useLoginMutation()`, `useLogoutMutation()`, `useChangePasswordMutation()`

- [ ] **Step 1: Add the test-utils helper (fresh `QueryClient` per test, per standard TanStack Query testing practice)**

`admin-ui/src/testUtils.tsx`:

```tsx
import type { ReactElement, ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderResult } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

export function renderWithQueryClient(
  ui: ReactElement,
  options: { route?: string; queryClient?: QueryClient } = {},
): RenderResult {
  const queryClient = options.queryClient ?? createTestQueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[options.route ?? "/"]}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(ui, { wrapper: Wrapper });
}
```

This is created now because it's the first task that needs a `QueryClientProvider` in tests; every later page/route test (Tasks 7–9) reuses it instead of redefining a wrapper.

- [ ] **Step 2: Write the failing tests for the auth query/mutation hooks**

`admin-ui/src/auth/queries.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { createTestQueryClient } from "../testUtils";
import { useChangePasswordMutation, useLoginMutation, useLogoutMutation, useWhoamiStatus } from "./queries";
import * as client from "../api/client";

vi.mock("../api/client");

afterEach(() => {
  vi.resetAllMocks();
});

function makeWrapper() {
  const queryClient = createTestQueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return Wrapper;
}

describe("useWhoamiStatus", () => {
  it("resolves to anonymous when whoami fails", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });

    const { result } = renderHook(() => useWhoamiStatus(), { wrapper: makeWrapper() });

    expect(result.current.status).toBe("loading");
    await waitFor(() => expect(result.current.status).toBe("anonymous"));
    expect(result.current.username).toBeNull();
  });

  it("resolves to authenticated when whoami succeeds", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    const { result } = renderHook(() => useWhoamiStatus(), { wrapper: makeWrapper() });

    await waitFor(() => expect(result.current.status).toBe("authenticated"));
    expect(result.current.username).toBe("furkan");
  });
});

describe("useLoginMutation", () => {
  it("updates the cached whoami status after a successful login, without a second whoami call", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });
    vi.mocked(client.login).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    const wrapper = makeWrapper();
    const { result: statusResult } = renderHook(() => useWhoamiStatus(), { wrapper });
    const { result: loginResult } = renderHook(() => useLoginMutation(), { wrapper });

    await waitFor(() => expect(statusResult.current.status).toBe("anonymous"));
    const whoamiCallsBeforeLogin = vi.mocked(client.whoami).mock.calls.length;

    loginResult.current.mutate({ username: "furkan", password: "s3cret-password" });

    await waitFor(() => expect(statusResult.current.status).toBe("authenticated"));
    expect(statusResult.current.username).toBe("furkan");
    expect(vi.mocked(client.whoami).mock.calls.length).toBe(whoamiCallsBeforeLogin);
  });

  it("leaves the cached whoami status anonymous when login fails", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });
    vi.mocked(client.login).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "invalid_credentials", message: "username or password is incorrect" },
    });

    const wrapper = makeWrapper();
    const { result: statusResult } = renderHook(() => useWhoamiStatus(), { wrapper });
    const { result: loginResult } = renderHook(() => useLoginMutation(), { wrapper });

    await waitFor(() => expect(statusResult.current.status).toBe("anonymous"));
    loginResult.current.mutate({ username: "furkan", password: "wrong" });

    await waitFor(() => expect(loginResult.current.data?.ok).toBe(false));
    expect(statusResult.current.status).toBe("anonymous");
  });
});

describe("useLogoutMutation", () => {
  it("updates the cached whoami status to anonymous after logout", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    vi.mocked(client.logout).mockResolvedValue({ ok: true, data: undefined });

    const wrapper = makeWrapper();
    const { result: statusResult } = renderHook(() => useWhoamiStatus(), { wrapper });
    const { result: logoutResult } = renderHook(() => useLogoutMutation(), { wrapper });

    await waitFor(() => expect(statusResult.current.status).toBe("authenticated"));
    logoutResult.current.mutate();

    await waitFor(() => expect(statusResult.current.status).toBe("anonymous"));
  });
});

describe("useChangePasswordMutation", () => {
  it("resolves ok on success", async () => {
    vi.mocked(client.changePassword).mockResolvedValue({ ok: true, data: undefined });

    const { result } = renderHook(() => useChangePasswordMutation(), { wrapper: makeWrapper() });
    result.current.mutate({ current_password: "old", new_password: "new-password-123" });

    await waitFor(() => expect(result.current.data?.ok).toBe(true));
  });
});
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
npm test -- src/auth/queries.test.tsx
```

Expected: FAIL — `./queries` does not exist yet.

- [ ] **Step 4: Implement the auth query/mutation hooks**

`admin-ui/src/auth/queries.ts`:

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  changePassword as apiChangePassword,
  login as apiLogin,
  logout as apiLogout,
  whoami as apiWhoami,
  type ApiResult,
  type WhoamiResponse,
} from "../api/client";

export const whoamiQueryKey = ["whoami"] as const;

export type AuthStatus = "loading" | "authenticated" | "anonymous";

export interface AuthStatusResult {
  status: AuthStatus;
  username: string | null;
}

export function useWhoamiStatus(): AuthStatusResult {
  const query = useQuery({
    queryKey: whoamiQueryKey,
    queryFn: apiWhoami,
  });

  if (query.isPending) {
    return { status: "loading", username: null };
  }
  if (query.data?.ok) {
    return { status: "authenticated", username: query.data.data.username };
  }
  return { status: "anonymous", username: null };
}

export function useLoginMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: apiLogin,
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.setQueryData<ApiResult<WhoamiResponse>>(whoamiQueryKey, {
          ok: true,
          data: { username: result.data.username },
        });
        toast.success("Signed in.");
      } else {
        toast.error(result.error.message);
      }
    },
  });
}

export function useLogoutMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: apiLogout,
    onSuccess: (result) => {
      if (result.ok) {
        queryClient.setQueryData<ApiResult<WhoamiResponse>>(whoamiQueryKey, {
          ok: false,
          status: 401,
          error: { code: "unauthorized", message: "not logged in" },
        });
        toast.success("Signed out.");
      } else {
        toast.error(result.error.message);
      }
    },
  });
}

export function useChangePasswordMutation() {
  return useMutation({
    mutationFn: apiChangePassword,
    onSuccess: (result) => {
      if (result.ok) {
        toast.success("Password updated.");
      } else {
        toast.error(result.error.message);
      }
    },
  });
}
```

Note on cache updates: `login`/`logout`/`changePassword` never *reject* (Task 4's `client.ts` always resolves an `ApiResult`, even on a 4xx), so `onSuccess` is where both the happy path and the API-error path are handled — there is no separate `onError` here for API-level failures, only for genuine network exceptions (which `mutationFn` would let propagate, entering TanStack Query's real `onError`/`isError` state; out of scope for this shell, same as the original plan's `AuthContext`).

- [ ] **Step 5: Run tests to verify they pass**

```bash
npm test -- src/auth/queries.test.tsx
```

Expected: PASS

- [ ] **Step 6: Wire `QueryClientProvider` and `<Toaster />` into the entrypoint**

Modify `admin-ui/src/main.tsx` (adds the query client and toaster on top of Task 3's font imports; Service Worker registration is added in Task 10):

```tsx
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/inter/600.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import App from "./App";
import "./index.css";

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("root element not found");
}

const queryClient = new QueryClient();

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      <Toaster
        theme="dark"
        toastOptions={{
          style: {
            background: "#171A21",
            color: "#F1F1F3",
            border: "1px solid #2A2E37",
          },
        }}
      />
    </QueryClientProvider>
  </StrictMode>,
);
```

`App` itself (Task 9) does **not** create its own `QueryClientProvider` — the client lives once at the true app root here in `main.tsx`, which is why `App.test.tsx` (Task 9) and every page test (Tasks 7–9) supply their own via `testUtils.tsx` when rendering `App`/pages in isolation.

- [ ] **Step 7: Run the full frontend test suite**

```bash
npm test
```

Expected: PASS (the Task 2 `App.test.tsx` smoke test still passes since `App.tsx` is unchanged until Task 9).

- [ ] **Step 8: Commit**

```bash
git add admin-ui/src/testUtils.tsx admin-ui/src/auth admin-ui/src/main.tsx
git commit -m "feat: add tanstack query provider and auth query/mutation hooks"
```

---

### Task 7: Login page (Formik + Yup + shadcn + TanStack Query + sonner)

**Files:**
- Create: `admin-ui/src/pages/LoginPage.tsx`
- Create: `admin-ui/src/pages/LoginPage.test.tsx`

**Interfaces:**
- Consumes: `useLoginMutation` (Task 6), `Button`/`Input`/`Label` (Task 5), `cn` (Task 5)
- Produces: `LoginPage` (default export component)

- [ ] **Step 1: Write the failing tests**

`admin-ui/src/pages/LoginPage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import LoginPage from "./LoginPage";
import { renderWithQueryClient } from "../testUtils";
import * as client from "../api/client";

vi.mock("../api/client");

function renderLoginPage() {
  return renderWithQueryClient(
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<p>Dashboard Home</p>} />
    </Routes>,
    { route: "/login" },
  );
}

afterEach(() => {
  vi.resetAllMocks();
});

describe("LoginPage", () => {
  it("submits the entered credentials and navigates on success", async () => {
    vi.mocked(client.login).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    renderLoginPage();

    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "furkan" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "s3cret-password" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() =>
      expect(client.login).toHaveBeenCalledWith({ username: "furkan", password: "s3cret-password" }),
    );
    await waitFor(() => expect(screen.getByText("Dashboard Home")).toBeInTheDocument());
  });

  it("shows a client-side validation error and never calls login when the password is left empty", async () => {
    renderLoginPage();

    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "furkan" } });
    fireEvent.blur(screen.getByLabelText("Password"));
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => expect(screen.getByText("Password is required.")).toBeInTheDocument());
    expect(client.login).not.toHaveBeenCalled();
  });

  it("shows the server error message on failed login", async () => {
    vi.mocked(client.login).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "invalid_credentials", message: "username or password is incorrect" },
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
import { useFormik } from "formik";
import * as Yup from "yup";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { useLoginMutation } from "../auth/queries";
import { cn } from "../lib/utils";

const loginSchema = Yup.object({
  username: Yup.string().required("Username is required."),
  password: Yup.string().required("Password is required."),
});

export default function LoginPage() {
  const navigate = useNavigate();
  const loginMutation = useLoginMutation();

  const formik = useFormik({
    initialValues: { username: "", password: "" },
    validationSchema: loginSchema,
    onSubmit: async (values) => {
      const result = await loginMutation.mutateAsync(values);
      if (result.ok) {
        navigate("/", { replace: true });
      }
    },
  });

  const serverError =
    loginMutation.data && !loginMutation.data.ok ? loginMutation.data.error.message : null;

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm rounded-lg border border-border bg-surface p-8 shadow-lg">
        <h1 className="mb-6 text-xl font-semibold text-text">Pixabros Admin</h1>
        <form onSubmit={formik.handleSubmit} className="space-y-4" noValidate>
          <div className="space-y-1.5">
            <Label htmlFor="username">Username</Label>
            <Input
              id="username"
              name="username"
              autoComplete="username"
              value={formik.values.username}
              onChange={formik.handleChange}
              onBlur={formik.handleBlur}
            />
            {formik.touched.username && formik.errors.username ? (
              <p className="text-sm text-error">{formik.errors.username}</p>
            ) : null}
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              value={formik.values.password}
              onChange={formik.handleChange}
              onBlur={formik.handleBlur}
            />
            {formik.touched.password && formik.errors.password ? (
              <p className="text-sm text-error">{formik.errors.password}</p>
            ) : null}
          </div>
          {serverError ? (
            <p
              role="alert"
              className="rounded-md border border-error/40 bg-error/10 px-3 py-2 text-sm text-error"
            >
              {serverError}
            </p>
          ) : null}
          <Button
            type="submit"
            disabled={formik.isSubmitting}
            className={cn("w-full", formik.isSubmitting && "opacity-70")}
          >
            {formik.isSubmitting ? "Signing in…" : "Sign in"}
          </Button>
        </form>
      </div>
    </main>
  );
}
```

`serverError` reads `loginMutation.data` directly rather than a separate local `useState` — because `useLoginMutation` never rejects on a 4xx (Task 6), `mutation.data` holds the last resolved `ApiResult` regardless of `ok`, so both success and failure are readable from the same reactive value without a redundant copy of it in component state. The inline `<p role="alert">` (asserted by tests) exists alongside — not instead of — the `toast.error(...)` call already wired into `useLoginMutation`'s `onSuccess` branch in Task 6.

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/pages/LoginPage.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/pages/LoginPage.tsx admin-ui/src/pages/LoginPage.test.tsx
git commit -m "feat: add login page with formik, yup, shadcn components, and tanstack query"
```

---

### Task 8: Change-password page (Formik + Yup + shadcn + TanStack Query + sonner)

**Files:**
- Create: `admin-ui/src/pages/ChangePasswordPage.tsx`
- Create: `admin-ui/src/pages/ChangePasswordPage.test.tsx`

**Interfaces:**
- Consumes: `useChangePasswordMutation` (Task 6), `Button`/`Input`/`Label` (Task 5), `cn` (Task 5)
- Produces: `ChangePasswordPage` (default export component)

- [ ] **Step 1: Write the failing tests**

`admin-ui/src/pages/ChangePasswordPage.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import ChangePasswordPage from "./ChangePasswordPage";
import { renderWithQueryClient } from "../testUtils";
import * as client from "../api/client";

vi.mock("../api/client");

afterEach(() => {
  vi.resetAllMocks();
});

describe("ChangePasswordPage", () => {
  it("shows a success message and clears the fields after a successful change", async () => {
    vi.mocked(client.changePassword).mockResolvedValue({ ok: true, data: undefined });

    renderWithQueryClient(<ChangePasswordPage />);

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "old-password-1" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "new-password-123" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(client.changePassword).toHaveBeenCalledWith({
        current_password: "old-password-1",
        new_password: "new-password-123",
      }),
    );
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Password updated."));
    expect(screen.getByLabelText("Current password")).toHaveValue("");
    expect(screen.getByLabelText("New password")).toHaveValue("");
  });

  it("shows a client-side validation error for a short new password and never calls the API", async () => {
    renderWithQueryClient(<ChangePasswordPage />);

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "old-password-1" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "short" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(screen.getByText("new password must be at least 8 characters")).toBeInTheDocument(),
    );
    expect(client.changePassword).not.toHaveBeenCalled();
  });

  it("shows the server-side weak_password error", async () => {
    vi.mocked(client.changePassword).mockResolvedValue({
      ok: false,
      status: 400,
      error: { code: "weak_password", message: "new password must be at least 8 characters" },
    });

    renderWithQueryClient(<ChangePasswordPage />);

    fireEvent.change(screen.getByLabelText("Current password"), { target: { value: "old-password-1" } });
    fireEvent.change(screen.getByLabelText("New password"), { target: { value: "longenough" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("new password must be at least 8 characters"),
    );
  });
});
```

The second test proves Yup's client-side rule (mirroring the backend's 8-character `weak_password` minimum) fires before any request goes out; the third test proves the real backend error still renders correctly on its own path — both are needed since one is UX-only and the other is the actual source of truth (per this plan's Global Constraints).

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm test -- src/pages/ChangePasswordPage.test.tsx
```

Expected: FAIL — `./ChangePasswordPage` does not exist yet.

- [ ] **Step 3: Implement the change-password page**

`admin-ui/src/pages/ChangePasswordPage.tsx`:

```tsx
import { useFormik } from "formik";
import * as Yup from "yup";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { useChangePasswordMutation } from "../auth/queries";
import { cn } from "../lib/utils";

const changePasswordSchema = Yup.object({
  current_password: Yup.string().required("Current password is required."),
  new_password: Yup.string()
    .required("New password is required.")
    .min(8, "new password must be at least 8 characters"),
});

export default function ChangePasswordPage() {
  const changePasswordMutation = useChangePasswordMutation();

  const formik = useFormik({
    initialValues: { current_password: "", new_password: "" },
    validationSchema: changePasswordSchema,
    onSubmit: async (values, helpers) => {
      const result = await changePasswordMutation.mutateAsync(values);
      if (result.ok) {
        helpers.resetForm();
      }
    },
  });

  const serverError =
    changePasswordMutation.data && !changePasswordMutation.data.ok
      ? changePasswordMutation.data.error.message
      : null;
  const succeeded = changePasswordMutation.data?.ok === true;

  return (
    <section className="mx-auto max-w-md">
      <h1 className="mb-6 text-xl font-semibold text-text">Change password</h1>
      <form
        onSubmit={formik.handleSubmit}
        className="space-y-4 rounded-lg border border-border bg-surface p-6"
        noValidate
      >
        <div className="space-y-1.5">
          <Label htmlFor="current-password">Current password</Label>
          <Input
            id="current-password"
            name="current_password"
            type="password"
            autoComplete="current-password"
            value={formik.values.current_password}
            onChange={formik.handleChange}
            onBlur={formik.handleBlur}
          />
          {formik.touched.current_password && formik.errors.current_password ? (
            <p className="text-sm text-error">{formik.errors.current_password}</p>
          ) : null}
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="new-password">New password</Label>
          <Input
            id="new-password"
            name="new_password"
            type="password"
            autoComplete="new-password"
            value={formik.values.new_password}
            onChange={formik.handleChange}
            onBlur={formik.handleBlur}
          />
          {formik.touched.new_password && formik.errors.new_password ? (
            <p className="text-sm text-error">{formik.errors.new_password}</p>
          ) : null}
        </div>
        {serverError ? <p role="alert" className="text-sm text-error">{serverError}</p> : null}
        {succeeded ? <p role="status" className="text-sm text-success">Password updated.</p> : null}
        <Button
          type="submit"
          disabled={formik.isSubmitting}
          className={cn("w-full", formik.isSubmitting && "opacity-70")}
        >
          {formik.isSubmitting ? "Saving…" : "Save"}
        </Button>
      </form>
    </section>
  );
}
```

Field `name`s (`current_password`, `new_password`) match `ChangePasswordRequest`'s JSON keys exactly, so `formik.values` can be passed straight to `changePasswordMutation.mutateAsync(values)` with no remapping.

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm test -- src/pages/ChangePasswordPage.test.tsx
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add admin-ui/src/pages/ChangePasswordPage.tsx admin-ui/src/pages/ChangePasswordPage.test.tsx
git commit -m "feat: add change-password page with formik, yup, shadcn components, and tanstack query"
```

---

### Task 9: Routing — `ProtectedRoute`, `Shell`, `DashboardPage`, `App` wiring

**Files:**
- Create: `admin-ui/src/components/ProtectedRoute.tsx`
- Create: `admin-ui/src/components/ProtectedRoute.test.tsx`
- Create: `admin-ui/src/pages/Shell.tsx`
- Create: `admin-ui/src/pages/Shell.test.tsx`
- Create: `admin-ui/src/pages/DashboardPage.tsx`
- Modify: `admin-ui/src/App.tsx`
- Modify: `admin-ui/src/App.test.tsx`

**Interfaces:**
- Consumes: `useWhoamiStatus`, `useLogoutMutation` (Task 6), `LoginPage` (Task 7), `ChangePasswordPage` (Task 8)
- Produces: `ProtectedRoute`, `Shell`, `DashboardPage`, the real `App` route tree (basename `/I-am-a-pixabro`)

- [ ] **Step 1: Write the failing test for `ProtectedRoute`**

`admin-ui/src/components/ProtectedRoute.test.tsx`:

```tsx
import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import { ProtectedRoute } from "./ProtectedRoute";
import { renderWithQueryClient } from "../testUtils";
import * as client from "../api/client";
import type { ApiResult, WhoamiResponse } from "../api/client";

vi.mock("../api/client");

afterEach(() => {
  vi.resetAllMocks();
});

function renderGuarded() {
  return renderWithQueryClient(
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
    </Routes>,
    { route: "/" },
  );
}

describe("ProtectedRoute", () => {
  it("shows a loading indicator while the whoami query is pending", () => {
    vi.mocked(client.whoami).mockReturnValue(new Promise<ApiResult<WhoamiResponse>>(() => {}));
    renderGuarded();
    expect(screen.getByRole("status")).toHaveTextContent("Checking session…");
  });

  it("redirects to /login when anonymous", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });
    renderGuarded();
    await waitFor(() => expect(screen.getByText("Login Screen")).toBeInTheDocument());
  });

  it("renders children when authenticated", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    renderGuarded();
    await waitFor(() => expect(screen.getByText("Protected Content")).toBeInTheDocument());
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
import { useWhoamiStatus } from "../auth/queries";

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { status } = useWhoamiStatus();

  if (status === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <p role="status" className="text-sm text-muted">
          Checking session…
        </p>
      </div>
    );
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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";
import Shell from "./Shell";
import { renderWithQueryClient } from "../testUtils";
import * as client from "../api/client";

vi.mock("../api/client");

afterEach(() => {
  vi.resetAllMocks();
});

function renderShell() {
  return renderWithQueryClient(
    <Routes>
      <Route path="/" element={<Shell />}>
        <Route index element={<p>Dashboard Body</p>} />
      </Route>
    </Routes>,
    { route: "/" },
  );
}

describe("Shell", () => {
  it("shows the signed-in username, nav links, and the routed dashboard body", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    renderShell();

    await waitFor(() => expect(screen.getByText("Signed in as furkan")).toBeInTheDocument());
    expect(screen.getByRole("link", { name: "Dashboard" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Change password" })).toBeInTheDocument();
    expect(screen.getByText("Dashboard Body")).toBeInTheDocument();
  });

  it("calls the logout endpoint when Log out is clicked", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });
    vi.mocked(client.logout).mockResolvedValue({ ok: true, data: undefined });

    renderShell();

    await waitFor(() => expect(screen.getByText("Signed in as furkan")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Log out" }));

    await waitFor(() => expect(client.logout).toHaveBeenCalled());
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
import { useLogoutMutation, useWhoamiStatus } from "../auth/queries";
import { cn } from "../lib/utils";

export default function Shell() {
  const { username } = useWhoamiStatus();
  const logoutMutation = useLogoutMutation();

  return (
    <div className="min-h-screen bg-background text-text">
      <header className="flex items-center justify-between border-b border-border bg-surface px-6 py-4">
        <span className="text-sm text-muted">Signed in as {username}</span>
        <nav className="flex items-center gap-4 text-sm">
          <Link to="/" className="text-text transition-colors hover:text-accent">
            Dashboard
          </Link>
          <Link to="/change-password" className="text-text transition-colors hover:text-accent">
            Change password
          </Link>
        </nav>
        <button
          type="button"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
          className={cn(
            "rounded-md border border-border px-3 py-1.5 text-sm text-text transition-colors hover:bg-background",
            logoutMutation.isPending && "opacity-50",
          )}
        >
          {logoutMutation.isPending ? "Logging out…" : "Log out"}
        </button>
      </header>
      <main className="px-6 py-8">
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
      <h1 className="text-2xl font-semibold text-text">Dashboard</h1>
      <p className="mt-2 text-sm text-muted">
        Module screens (Games, Members, Devlog, Awards, Contact, Site Settings, Media) land here in later plans.
      </p>
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
import { ProtectedRoute } from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import Shell from "./pages/Shell";
import DashboardPage from "./pages/DashboardPage";
import ChangePasswordPage from "./pages/ChangePasswordPage";

export default function App() {
  return (
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
  );
}
```

`App` has no `QueryClientProvider` of its own — that lives once at the real app root in `main.tsx` (Task 6). Tests that render `App` directly must supply their own, as below.

- [ ] **Step 10: Rewrite `App.test.tsx` to cover both auth states**

`App` renders a `BrowserRouter` with `basename="/I-am-a-pixabro"`, but jsdom's default test location is `/`, which does not start with that basename — react-router would fail to match any route. Point the test's location at the basename before each render and restore it afterward. Also wrap in a fresh `QueryClient` per test, since `App` no longer provides one itself.

`admin-ui/src/App.test.tsx`:

```tsx
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import App from "./App";
import { createTestQueryClient } from "./testUtils";
import * as client from "./api/client";

vi.mock("./api/client");

beforeEach(() => {
  window.history.pushState({}, "", "/I-am-a-pixabro/");
});

afterEach(() => {
  vi.resetAllMocks();
  window.history.pushState({}, "", "/");
});

function renderApp() {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

describe("App", () => {
  it("shows the login page when anonymous", async () => {
    vi.mocked(client.whoami).mockResolvedValue({
      ok: false,
      status: 401,
      error: { code: "unauthorized", message: "not logged in" },
    });

    renderApp();

    await waitFor(() => expect(screen.getByRole("button", { name: "Sign in" })).toBeInTheDocument());
  });

  it("shows the dashboard when authenticated", async () => {
    vi.mocked(client.whoami).mockResolvedValue({ ok: true, data: { username: "furkan" } });

    renderApp();

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

- [ ] **Step 12: Run lint**

```bash
npm run lint
```

Expected: zero errors, in particular zero `@typescript-eslint/no-explicit-any` violations.

- [ ] **Step 13: Commit**

```bash
git add admin-ui/src/App.tsx admin-ui/src/App.test.tsx admin-ui/src/components admin-ui/src/pages/Shell.tsx admin-ui/src/pages/Shell.test.tsx admin-ui/src/pages/DashboardPage.tsx
git commit -m "feat: add routing, protected-route guard, shell, and dashboard placeholder"
```

---

### Task 10: Service Worker for app-shell asset caching

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

Modify `admin-ui/src/main.tsx` (final version — Task 3's font imports, Task 6's `QueryClientProvider`/`Toaster`, and this task's Service Worker registration all together):

```tsx
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/inter/600.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";
import App from "./App";
import { registerServiceWorker } from "./registerServiceWorker";
import "./index.css";

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("root element not found");
}

const queryClient = new QueryClient();

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      <Toaster
        theme="dark"
        toastOptions={{
          style: {
            background: "#171A21",
            color: "#F1F1F3",
            border: "1px solid #2A2E37",
          },
        }}
      />
    </QueryClientProvider>
  </StrictMode>,
);

registerServiceWorker();
```

- [ ] **Step 7: Manual browser verification**

`jsdom` has no `ServiceWorkerGlobalScope`, so the SW's `fetch`-interception behavior cannot be unit tested — verify it manually once the shell is built and served (after Task 12's server is running):

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

### Task 11: Build output wiring into the Go server's expected directory

**Files:**
- Create: `Makefile`
- Modify: `.gitignore` (repo root, from Plan A)

**Interfaces:**
- Consumes: `admin-ui`'s `npm run build` (Task 2's `vite.config.ts` `outDir` logic; Task 3's Tailwind/PostCSS config runs automatically as part of the same `vite build` — there is no separate CSS build step to wire in)
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

`npm run build` is `tsc -b && vite build` (Task 2's `package.json`); Vite's own PostCSS pipeline picks up `postcss.config.js`/`tailwind.config.js` (Task 3) automatically for any imported CSS, so `make admin-build` alone produces fully Tailwind-compiled CSS — there is no separate "run Tailwind" step to remember or forget.

- [ ] **Step 2: Extend the root `.gitignore`**

Append to `.gitignore` (from Plan A Task 1 — it already ignores `/data/`, which covers the build *output* since it lands under `./data/admin-dist`; this adds the frontend's own build tooling artifacts):

```gitignore
/admin-ui/node_modules/
```

- [ ] **Step 3: Run the build and verify the output lands in the right place**

```bash
make admin-build
ls data/admin-dist
ls data/admin-dist/assets
```

Expected: `data/admin-dist/index.html`, `data/admin-dist/sw.js`, and `data/admin-dist/assets/` (containing content-hashed `.js`/`.css` files) all exist.

- [ ] **Step 4: Verify the base path is baked into the built HTML**

```bash
grep -o '/I-am-a-pixabro/assets/[^"]*\.js' data/admin-dist/index.html
```

Expected: at least one match — confirms Vite's `base: "/I-am-a-pixabro/"` (Task 2 Step 6) was applied to the production build's asset URLs.

- [ ] **Step 5: Verify Tailwind's compiled CSS is part of the same build output**

```bash
grep -il "0f1115" data/admin-dist/assets/*.css
grep -il "e879f9" data/admin-dist/assets/*.css
```

Expected: both design-spec hex tokens are present in the same `assets/*.css` file that `make admin-build` produced — proof that Tailwind/PostCSS ran inside `npm run build`, not as a separate manual step.

- [ ] **Step 6: Commit**

```bash
git add Makefile .gitignore
git commit -m "feat: add make target building the admin ui into the server's expected directory"
```

---

### Task 12: End-to-end verification against the Go server

**Files:**
- None (verification only)

**Interfaces:**
- Consumes: everything from Tasks 1–11

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

Expected: login returns `200` with `Set-Cookie: pixabros_session=...; HttpOnly; Secure; SameSite=Strict` and body `{"username":"furkan"}`; the second `whoami` call returns `200` with the same body; the `sw.js` request returns `200` with the Service Worker source from Task 10.

- [ ] **Step 5: Verify the change-password and logout flow**

```bash
curl -i -b /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/change-password \
  -H 'Content-Type: application/json' \
  -d '{"current_password":"a-strong-password-1","new_password":"a-different-strong-password-2"}'
curl -i -b /tmp/pixabros-cookies.txt -X POST http://localhost:8080/api/admin/logout
curl -i -b /tmp/pixabros-cookies.txt http://localhost:8080/api/admin/whoami
```

Expected: change-password returns `204`; logout returns `204` and clears the cookie; the final `whoami` call (cookie now invalid) returns `401`.

- [ ] **Step 6: Open the SPA in a real browser and click through it, confirming the design spec's visuals and toasts**

```bash
open http://localhost:8080/I-am-a-pixabro/
```

Walk through, confirming at each step:
- **Login screen:** dark background (`#0F1115`), a card surface (`#171A21`) with a visible border, Inter-rendered text (not a system-font fallback — check via DevTools → Elements → Computed → `font-family` on the heading), and an accent-magenta (`#E879F9`) "Sign in" button that visibly darkens (`#C026D3`) on hover.
- Log in with `furkan` / `a-different-strong-password-2`; confirm a **sonner toast** ("Signed in.") appears in the corner, styled with the dark surface/border/text colors from Task 6's `<Toaster />` config, and the dashboard placeholder renders with "Signed in as furkan".
- Navigate to "Change password", submit a new password; confirm both the inline "Password updated." status text **and** a success toast appear.
- Click "Log out"; confirm a "Signed out." toast appears and you land back on the login screen.
- Complete Task 10 Step 7's DevTools Service Worker check here too (scope `/I-am-a-pixabro/`, hashed assets served `(ServiceWorker)` on reload, `/api/admin/*` never served from cache).

- [ ] **Step 7: Stop the server and run the full test suites one last time**

```bash
kill %1
go test ./...
(cd admin-ui && npm test)
(cd admin-ui && npm run lint)
```

Expected: PASS for every Go package, every Vitest file, and zero lint errors.

- [ ] **Step 8: Final commit (only if any stray files remain from verification)**

```bash
git status
```

If everything from Tasks 1–11 is already committed, there is nothing left to commit here — this task is verification-only.

---

## Definition of Done

- `go build ./...` and `go test ./...` succeed, including the new `Whoami` handler and router tests.
- `cd admin-ui && npm test` passes with no skipped files — covering the typed API client, the TanStack Query auth hooks (`useWhoamiStatus`/`useLoginMutation`/`useLogoutMutation`/`useChangePasswordMutation`), the shadcn `Button`/`Input`/`Label` primitives, both Formik+Yup forms (client-side Yup errors *and* server-side `ApiError` messages asserted separately), `ProtectedRoute`, `Shell`, `App`, and the Service Worker registration logic.
- `npm run lint` reports zero errors, in particular zero `@typescript-eslint/no-explicit-any` violations.
- `make admin-build` produces `data/admin-dist/index.html` + a content-hashed `assets/` bundle (containing compiled Tailwind CSS with the design spec's exact color tokens) + `sw.js`, with `/I-am-a-pixabro/` baked into every asset URL — all from a single `npm run build`, with no separate manual Tailwind step.
- Against a running `cmd/server`: an anonymous `GET /api/admin/whoami` returns `401`; `POST /api/admin/login` with valid credentials sets the `pixabros_session` cookie and a subsequent `whoami` returns `200` with the username; `POST /api/admin/change-password` and `POST /api/admin/logout` behave per their Plan A contracts; the logged-out session's `whoami` call returns `401` again.
- In a real browser at `http://localhost:8080/I-am-a-pixabro/`: login → dashboard → change-password → logout all work end-to-end; the dark theme, accent color (default and hover), and Inter font are visually confirmed; a sonner toast appears on login, change-password, and logout; and DevTools confirms the Service Worker is active with scope `/I-am-a-pixabro/`, serves hashed JS/CSS from cache on repeat loads, and never serves `/api/*` from cache.
