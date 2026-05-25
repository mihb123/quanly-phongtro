## Core Working Rules
> Rules are ordered by impact on code quality. When trade-offs arise, higher-priority rules win.

---

### 🔴 Critical — Directly affects architecture, correctness, and logic quality

### 1) Separation of Concerns
- Enforce strict separation of concerns at the file level.
- Do not combine different layers of responsibility (e.g. UI, state orchestration, business logic, data access) into a single file.
- Split distinct logic into dedicated files and directories.

### 2) Function Design
- Keep functions focused on one job.
- If validation, orchestration, persistence, and response shaping start to mix, split the logic by responsibility.

### 3) Control Flow and Data Flow
- Keep control flow shallow. Prefer early returns over deep nesting.
- Keep data flow explicit. Avoid hidden mutation, broad shared state, and magic values inside business logic.
- Avoid boolean parameters that change function behavior in non-trivial ways. Prefer separate functions or named options when that makes intent clearer.

### 4) Error Handling
- Never ignore returned errors.

---

### 🟡 Important — Affects consistency and long-term maintainability

### 5) Reuse Existing Project Patterns
- Reuse existing patterns, helpers, API functions, services, and naming from the touched module before creating new ones.

### 6) Code Style and Naming
- Add a short comment before every function that explains the function responsibility, constraint, or intent. (just func you add)
- Use descriptive names. Avoid abbreviations like `userRepo`; prefer `userRepository`. But avoid long names.
- Reuse existing domain terms in the file when possible.
- Always write code in the current style of the project.
- If this is an initial project, skip this rule and create the project following the rules.

### 7) Clarify Before Changing
- If the request is ambiguous, state the assumptions and ask before editing. Do not silently choose between multiple valid interpretations.

---

### 🟢 Standard — Process, scope control, and code hygiene

### 8) Keep Changes Surgical
- Touch only files directly related to the request. Do not refactor, rename, or reformat unrelated code.
- Prefer the simplest implementation that solves the requested behavior. Do not add speculative abstractions, configuration, or edge-case handling that was not asked for.

### 9) Cleanup After Changes
- Remove imports, variables, and helpers made unused by your own change. Do not delete unrelated dead code; mention it separately.

### 10) Validation and Delivery
- Do not run `migrate`, `build`, or deploy commands unless explicitly requested. If validation depends on them, tell the user what to run.
- Write simple and maintainable code. Consider business perspective.

## Frontend Rules (React)
- Do not place page rendering, data fetching, state orchestration, and helper logic in one file.
- Enforce strict layer purposes: Put UI pieces in `@/components`, HTTP calls in `@/api`, reusable stateful logic in `@/hooks`, pure helpers in `@/utils`, global domain data and CRUD operations in `@/data` (Zustand stores), and shared low-level helpers in `@/lib`.
- Keep page-level files thin: compose components, call hooks, and import API or utility helpers instead of reimplementing them inline.
- All HTTP calls must go through the shared API layer in `@/api`. Never call `fetch` or any HTTP client directly in components, hooks, or pages.
- Reuse existing API functions before adding new ones. Do not duplicate the same request logic in multiple files.
- Do not store state that can be derived from props, query data, or other state.
- When a screen grows beyond a small feature slice, split view, hook, API, and helper logic early.
- Wrap components that receive array or object props with `React.memo` when the parent re-renders frequently. Pass callbacks through `useCallback` and derived data through `useMemo`.
- Avoid `dangerouslySetInnerHTML`. If HTML injection is required, sanitize or escape the content first and document the XSS-safe guarantee in a comment.
- All modals MUST close on Escape keypress and backdrop click. Always ensure consistent implementation across all newly created or updated modals.
- All interactive elements (buttons, icon buttons, clickable icons, links, and clickable areas) MUST show a pointer cursor (`cursor-pointer` class). Ensure that generic UI components (like `<Button>`) have this class set by default in their base variants. Any async action triggered from the UI must expose a visible loading state.
- If a component is the only place that triggers a modal, that component should own the modal state and render the modal itself. Do not lift modal state to a parent just to pass callbacks back down.
- Modals must be self-contained: they should receive minimal props (e.g., `onClose`, and an `id` or entity if not available from global state). The modal itself should handle its own API calls (via `@/api` or Zustand stores) and not rely on the parent for `onSuccess` callbacks.
- Use global stores (Zustand in `@/data`) for shared domain data and CRUD orchestration that multiple components read (e.g., houses, rooms, selected state). Use `getState()` for cross-store reads when possible instead of subscribing. Keep UI-specific state (modal visibility, selection, loading states) local to the owning component.
- Always use React Hook Form + Zod for all form-based modals and pages. Avoid using manual `useState` for individual form fields to prevent unnecessary re-renders on every keystroke. Only use `useState` for complex custom inputs (like File uploads or image previews) that RHF doesn't handle natively.
- Moving `useState` from a component into a custom hook does not reduce re-renders — it only moves code. Only extract hooks when the logic is reusable across multiple components or when separation of concerns demands it.
- Do not create abstractions to eliminate prop passing that is only 1 level deep. Optimize prop drilling only when data flows through 2 or more intermediate components that do not use it.

## Backend Rules (Go)
- Do not combine routing, request parsing, middleware, business logic, database access, and response shaping in one file.
- Keep `routes` responsible only for route registration, `handler` for HTTP input and output, `middleware` for cross-cutting request concerns, `service` for business rules, `repository` for database queries, and `dto` for request and response contracts.
- Handlers must stay thin: validate input, call a service, and return a DTO or error response. They must not contain query logic, data joining between collections, or long business flows.
- Repositories are the only layer that talks directly to the database. Services may coordinate multiple repositories, but should not build HTTP responses.
- Prefer named structs over long multi-value returns and over anonymous structs inside functions.
- Define all struct types at the top of the file, before functions.
- Do not leak database models directly to API responses. Map data through DTOs or explicit response structs.
- Use existing response and error conventions already present in the module. Do not introduce a new response shape for one endpoint unless the task requires it.
- Configurable values used in business logic must come from the config layer, not from hardcoded service-level variables.
- Cache implementations must have a bounded size or a cleanup mechanism. Do not introduce maps that only grow.
- Tell me if you find dead or unreachable code.