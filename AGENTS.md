# Drive — Development Rules

## Test-Driven Development

Every change starts with a failing test. No feature or fix is accepted without a corresponding test diff. Tests must run as a single command and complete in under 10 seconds.

## Test Layering

### Backend (Go)
1. **Store tests** — Real `:memory:` SQLite via `testutil.OpenTestDB()`. Migrations run, every CRUD method exercised. No mocking.
2. **Handler tests** — `httptest.NewServer` with real chi router and `:memory:` DB. Assert status codes and JSON response shapes.
3. **Worker tests** — Unit-test the worker pool with `:memory:` store. Inject jobs, verify state transitions. Use `-race` flag.
4. **Integration tests** — Full roundtrip: create user -> login -> upload -> list -> download.

### Frontend (TypeScript/Vue)
1. **Store tests** — Pinia stores with mock API client. Assert state transitions.
2. **Component tests** — `@vue/test-utils` to mount, trigger events, assert DOM. Mock API calls.
3. **Router tests** — Navigation guards with auth state permutations.

## Test Naming

- File: `{source}_test.go` or `{source}.test.ts` (colocated)
- Function: `Test{Pkg}_{Unit}_{scenario}` e.g. `TestUserStore_Create_shouldReturnUser`
- Table-driven tests where multiple inputs share logic

## Coverage Targets

| Layer | Target |
|---|---|
| Store | 90%+ |
| Handler/Middleware | 70%+ |
| Worker | 80%+ |
| Config | 80%+ |
| Frontend stores | 90%+ |
| Frontend components | 60%+ |

## Required Before Finishing

**Before any piece of work is considered complete, you MUST run `make test-all` and ensure all existing tests pass. Any new feature or fix MUST include one or more new tests that fail without the change and pass with it.** Never skip running the test suite — even for small or "obvious" changes.

## Running Tests

```bash
make test          # go test -count=1 ./...
make test-cover    # go test -race -count=1 -coverprofile=coverage.out ./...
make test-web      # cd web && npx vitest run
make test-all      # both (test + test-web)
```

Backend tests run against `:memory:` SQLite with real migrations. Frontend tests run in jsdom with mocked API calls. The full suite completes in under 15 seconds.

## Code Style

- No comments unless explaining a non-obvious design decision
- Follow existing patterns in the codebase
- Imports grouped: stdlib, third-party, project
- Errors always wrapped with `fmt.Errorf("context: %w", err)`
- Use `interface{}` only for JSON map literals
