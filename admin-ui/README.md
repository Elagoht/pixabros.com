# IC Frontend Template

A reusable React + TypeScript frontend template with a comprehensive component library, built for internal panel applications.

## Tech Stack

| Category    | Technology                                  |
| ----------- | ------------------------------------------- |
| Framework   | React 19, Vite 8                            |
| Language    | TypeScript 6                                |
| Styling     | Tailwind CSS 3 (dark mode enabled)          |
| Routing     | react-router-dom v7 (`createBrowserRouter`) |
| State       | Zustand (global), React Query (server)      |
| Forms       | Formik 2 + Yup                              |
| Tables      | @tanstack/react-table 8                     |
| Icons       | @tabler/icons-react                         |
| Drag & Drop | @dnd-kit/core                               |
| Toasts      | sonner                                      |
| Compiler    | React Compiler enabled                      |

## Getting Started

```bash
make install
make dev        # Start dev server at http://localhost:5173
```

## Commands

```bash
make dev        # Start development server
make build      # Type-check + build for production
make install    # Install dependencies
make clean      # Remove node_modules
```

## Project Structure

```
src/
├── app/                  # Next.js-style file-based routing
│   ├── (auth)/           # Guest pages (login, register)
│   ├── (panel)/          # Authenticated pages
│   ├── demo/             # Component demo page (/demo)
│   └── not-found.tsx     # 404 page
├── components/            # Reusable components
│   ├── guards/           # Route guards (AuthGuard, GuestGuard)
│   └── ui/               # 70+ UI components
├── design/               # CSS (Tailwind directives)
├── forms/                # All Formik forms
├── hooks/                # Custom React hooks
├── i18n/                 # Translations (en, tr, ru)
├── lib/                  # Library wrappers & config
│   ├── query/            # React Query client & query keys
│   ├── routes/           # Router configuration
│   ├── stores/           # Zustand stores
│   └── validation/       # Yup schemas
├── pages/                # Page-specific components
├── services/             # API service classes
├── types/                # Global TypeScript declarations
└── utilities/            # Utility functions (math, color, http, date-locale, etc.)
```

## UI Components

The `src/components/ui/` directory contains 70+ production-ready components:

- **Forms**: Input, Select, Combobox, Checkbox, CheckboxGroup, RadioGroup, Switch, FileInput, DatePicker, DateTimePicker, PhoneInput, OTPInput, ColorPicker, RangeSlider, Rating, Keywords, Chip
- **Layout**: Container, PageLayout, Sidebar, Navbar, MenuBar, TOC, Drawer, ScrollArea
- **Data**: DataTable, HierarchyTable, CategoryTable, Pagination, Stepper
- **Feedback**: Alert, ProgressBar, ProgressCircle, Skeleton, Loading, EmptyState
- **Navigation**: Breadcrumb, Tabs, Timeline, Accordion, Carousel
- **Overlay**: Modal, Dialog, Dropdown, Popover, HoverCard, ContextMenu, CommandPalette, Tooltip
- **Content**: Card, Badge, Avatar, Button, ButtonGroup, Separator, Kbd, CodeBlock, CopyButton, Image, FAB, Label, FieldSet

All interactive components (Dropdown, Tabs, Accordion, ContextMenu) use **data-driven APIs** — pass arrays of items instead of nested JSX.

## Key Conventions

- **Arrow functions only.** `const foo = () => {}`
- **Classes for services/singletons.** `export class MyService { static method() {} }`
- All types in `src/types/*.d.ts` are global (no imports/exports)
- All user-facing text uses i18n (`useI18n()` or `t()`)
- One component per file, barrel-exported from `ui/index.ts`
- All Formik forms live in `src/forms/`, all Yup schemas in `src/lib/validation/`
- Dark mode is mandatory for every component
- `@/` path alias → `src/`
