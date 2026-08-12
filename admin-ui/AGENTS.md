# AGENTS.md — AI Coding Instructions

## Style Preferences
- **Arrow functions over `function` keyword.** Always use `const foo = () => {}` syntax. No exceptions.
- **Classes over object literals** for singletons/services. `export class MyService { static method() {} }`, not `export const myService = { method() {} }`.
- **Data-driven over compound components.** Prefer passing `items` arrays to components instead of nesting `<Component.Sub>...</Component.Sub>` JSX.

## Component Usage — CRITICAL
- **Before writing ANY JSX, check `COMPONENTS.md`.** It lists all 70+ available components, their imports, and their props.
- **NEVER create ad-hoc `button`, `input`, `select`, or `textarea` elements.** Use Button, Input, Select, Textarea.
- **NEVER create ad-hoc `div`/`span` for UI patterns** that have existing components (Badge, Avatar, Skeleton, Separator, Kbd, Chip, etc.).
- **NEVER create custom modals, toasts, tooltips, or dropdowns.** Use Modal/Dialog/Drawer, sonner, Tooltip, Dropdown.
- Import all components from `@/components/ui`. One import statement: `import { Button, Card, Input } from "@/components/ui";`

## Type System
- **NEVER use `any` type.** Always define proper TypeScript types.
- **Global types** live in `src/types/*.d.ts` as ambient declarations. **Do NOT use `import` or `export`** in these files. Types declared here are available globally without imports.
- If you need to reference an external type in a `.d.ts` file, use `declare` blocks.

## Component Patterns
- All UI components go in `src/components/ui/`. Barrel-exported from `src/components/ui/index.ts`.
- **One component per file.** Single default export.
- Multi-file components use directories: `ComponentName/index.tsx`, `ComponentName/Helper.tsx`.
- Always support **dark mode** via `dark:` Tailwind variants.
- Use `classnames` for conditional classes.
- Use `@tabler/icons-react` for icons. Icon prop type is `IconElement` (global type).
- Form components use `useField` from Formik. Context is provided via `FormikProvider`.
- Generic arrow components in `.tsx` use `<T,>` trailing comma syntax to disambiguate from JSX.

## Forms
- **ALL Formik forms go in `src/forms/`.** Pages only import and render.
- **ALL Yup validation schemas go in `src/lib/validation/`.** Use arrow functions that accept `t()` for i18n.

## i18n
- Translation files: `src/i18n/{en,tr,ru}.json`. **All 3 must share identical key structure.**
- Use `useI18n()` hook or `t()` for all user-facing text.
- Never hardcode strings in components.

## Routing
- Routes defined in `src/lib/routes/index.tsx` using `createBrowserRouter`.
- Pages use Next.js-style file naming in `src/app/`: `layout.tsx`, `page.tsx`, `not-found.tsx`.
- Route groups: `(auth)` for guest pages, `(panel)` for authenticated pages.

## State
- **Zustand** for global state. Stores in `src/lib/stores/`.
- **React Query** for server state. Client in `src/lib/query/client.ts`.
- Prefer prop drilling over unnecessary global state.

## Services
- Service classes in `src/services/`. One class per file. Static methods or singleton instances.

## Styling
- **Tailwind CSS** only. No CSS modules, no inline styles except dynamic values.
- Custom colors: `primary` (warm brown/gold), `secondary` (blue), `gray` (warm gray).
- Border radius: `rounded-md` default, `rounded-lg` for cards/containers.

## No Comments
- Do NOT add comments to code unless explicitly requested.

## Build Commands
- `make dev` — start dev server
- `make build` — type-check + build
- `make check` — biome lint + format check
- `make format` — biome format auto-fix
- `make lint` — biome lint
- `make install` — install dependencies
- `make clean` — remove node_modules
- `make test` — run all tests
- `make test-watch` — run tests in watch mode
- `make storybook` — start Storybook dev server

## Testing
- **Vitest** + **jsdom** environment. Config in `vitest.config.ts`.
- Test files in `test/` directory. Run with `npm test` or `make test`.
- **@testing-library/react** for component testing. **@testing-library/user-event** for interactions.
- Formik components must be wrapped in `<FormikProvider>` in tests.
- Route-dependent components must be wrapped in `<MemoryRouter>` in tests.
- Mock external modules (`@/utilities/http`, `@/services/session`, `@/lib/stores/auth`) with `vi.mock()`.

## Pre-commit Hooks
- **Husky** + **lint-staged** runs on every commit.
- lint-staged runs `biome check --write --no-errors-on-unmatched` on `*.{ts,tsx}` files.
- Tests are NOT run on pre-commit to keep commits fast.
