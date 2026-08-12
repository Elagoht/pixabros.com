# Contribution Guide

## Project Structure

```
src/
├── app/                  # Next.js-style file-based routing
│   ├── (auth)/           # Auth route group (GuestGuard)
│   │   ├── layout.tsx    # Auth layout
│   │   └── login/
│   │       └── page.tsx  # Login page → imports from src/forms/
│   ├── (panel)/          # Panel route group (AuthGuard)
│   │   ├── layout.tsx    # Panel layout
│   │   └── ...
│   ├── demo/
│   │   └── page.tsx      # Component demo page
│   └── not-found.tsx     # 404 page
├── components/            # All reusable components
│   ├── guards/           # AuthGuard, GuestGuard
│   └── ui/               # UI component library (70+ components)
├── design/               # CSS files (Tailwind directives)
├── forms/                # ALL Formik forms - NO EXCEPTIONS
├── hooks/                # Custom React hooks
├── i18n/                 # Translation JSON files (en, tr, ru)
├── lib/                  # Third-party library wrappers
│   ├── query/            # React Query client & keys
│   ├── routes/           # react-router-dom router config
│   ├── stores/           # Zustand stores (auth, i18n)
│   └── validation/       # ALL Yup validation schemas - NO EXCEPTIONS
├── pages/                # Page-specific components
│   └── {same-path-as-app}/
├── services/             # Service classes (one class per file)
├── types/                # Global .d.ts type declarations
└── utilities/            # In-house utility functions
```

## Rules

### Style
- **Arrow functions only.** `const foo = () => {}` — never `function foo() {}`.
- **Classes over objects.** For singletons/services: `export class MyService { static method() {} }` — never `export const myService = { method() {} }`.
- **Data-driven components.** Prefer passing arrays to components via props over compound JSX nesting.

### src/forms/
All Formik forms go here. One form per file. Pages must import from this directory. **No Formik form definitions allowed inside `page.tsx` files.**

### src/app/
- Uses Next.js-style convention: `(group)`, `[param]`, `layout.tsx`, `page.tsx`, `not-found.tsx`
- Every route must be registered in `src/lib/routes/index.tsx`
- `page.tsx` files must NOT contain component definitions — import from `src/forms/` or `src/components/`

### src/components/
- One component per file. Single default export.
- Multi-component patterns use subdirectories: `ComponentName/index.tsx`, `ComponentName/SubComponent.tsx`

### src/pages/
Page-specific components mirror the `src/app/` path structure.

### src/types/
- **NEVER use `import` or `export`** in `.d.ts` files
- All types are declared as ambient global declarations
- If you must reference an external type, use `declare`

### src/i18n/
- All 3 JSON files must share the **exact same schema**
- **Every key in `en.json` must exist in `tr.json` and `ru.json`**
- All text in components and pages must be translatable via `t()` or `useI18n()`
- `TranslationKey` auto-generated type provides full IDE autocomplete

### src/lib/validation/
- All Yup validation schemas go here
- Factory arrow functions: `export const loginSchema = (t) => Yup.object({...})`

### src/services/
- Each service is a class in its own file
- Named export: `export class MyService { ... }` or `export class myService { static ... }`

### src/hooks/
- Custom hooks only. `use-` prefix recommended.
- Arrow functions: `export const useHook = () => { ... }`

## Tech Stack
- React 19, TypeScript 6, Vite 8
- Tailwind CSS 3 — all styling via utility classes
- react-router-dom v7 — `createBrowserRouter`
- Zustand for global state, react-query for server state
- Formik + Yup for forms
- @tabler/icons-react for icons
- sonner for toasts
- React Compiler enabled

## Conventions
- Path alias `@/` → `src/`
- Dark mode: every component must support `dark:` variants
- Use `classnames` for conditional classes
- ABSOLUTELY NO `any` type — use proper TypeScript types
- All components default-exported, barrel re-exported from `src/components/ui/index.ts`
- Run `make check` before committing to ensure formatting and linting pass

## Formatting & Linting
- **Biome.js** for formatting + linting — replaces Prettier + ESLint entirely
- Format: `make format` / Check: `make check` / Lint: `make lint`
- Config: `biome.json` — strict rules for const, block statements, equality, no explicit any, no var, no forEach
- Git blame: `.git-blame-ignore-revs` — formatting commits won't pollute blame history
- Config: `biome.json` — strict rules for const, block statements, equality, no explicit any, no var, no forEach
- Git blame: `.git-blame-ignore-revs` — formatting commits won't pollute blame history
