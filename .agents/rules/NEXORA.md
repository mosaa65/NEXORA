# NEXORA Development Rules

> These rules guide both human developers and AI assistants working on the NEXORA project.

## Project Overview
NEXORA is a LAN-first media management and streaming system with:
- **Backend**: Go REST API with PostgreSQL + Meilisearch
- **Frontend**: React 19 + Vite + TailwindCSS (RTL-first, Arabic UI)
- **Architecture**: Client-server with admin portal isolation

## Design System Rules

### 1. Design Tokens Only
- **NEVER** use hardcoded colors like `#7C3AED` in components
- **ALWAYS** use CSS custom properties: `var(--color-primary)`, `var(--bg-base)`, etc.
- All tokens are defined in `client/src/design-system/tokens.css`

### 2. Component Library
- Reusable UI primitives live in `client/src/components/ui/`
- Always import from `components/ui/` before creating inline styles
- Available components: `Button`, `Input`, `Badge`, `Card`, `Modal`, `ProgressBar`, `Spinner`

### 3. Theme System
- Support dark (default) and light themes via `data-theme` attribute
- Theme switching handled by `ThemeContext`
- All theme overrides go in `client/src/design-system/themes/`

## File Structure Rules

### 4. No Giant Files
- Maximum **300 lines** per component file
- Split large pages into sub-components
- Admin pages go in `client/src/pages/admin/`
- Customer pages go in `client/src/pages/customer/` or `client/src/pages/`

### 5. Component Organization
```
components/
├── ui/        → Primitive components (Button, Input, Card, Modal)
├── media/     → Media-specific (MediaCard, HeroSlider, VideoPlayer)
├── layout/    → Structural (Sidebar, TopBar)
└── admin/     → Admin-specific (DiskCard, IndexerPanel)
```

### 6. Documentation
- All docs live in `docs/` with subdirectories by topic
- Old files are archived in `_archive/` (not deleted)
- `CONTRIBUTING.md` in root for team onboarding

## Code Style Rules

### 7. Language
- Code comments and variable names: **English**
- UI text displayed to users: **Arabic** (RTL)
- Use `dir="rtl"` on root containers

### 8. Imports
- Prefer named imports from barrel files (`components/ui/index.js`)
- Use relative imports within the same feature folder
- API calls go through `lib/api.js`

### 9. State Management
- Use React Context for global state (themes, auth)
- Use local `useState` for component-specific state
- No external state libraries (Redux, Zustand) needed

## Git Rules

### 10. Never Commit
- Executable files (`*.exe`, `*.exe~`)
- `node_modules/`, `dist/`, `tmp/`
- Personal tool configs (`.codebuddy/`, `.cache/`, `.logs/`)
- Environment files (`.env`)

### 11. Commit Messages
Follow Conventional Commits: `type(scope): description`
- Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
