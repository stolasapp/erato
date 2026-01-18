# Erato - Project Context for Claude

## Overview

Erato is an archive proxy and tracking tool that acts as an intermediary between users and a custom/proprietary upstream archive service.
It provides:

- A proxy layer that scrapes and normalizes archive content
- User interaction tracking (read times, bookmarks, starred items)
- Both a ConnectRPC API and a web frontend for browsing archives
- Hierarchical content structure: Categories → Entries → Chapters

This is a **v2 major refactor** with a complete redesign of the architecture, UI, API, and database schema.

## Development Environment

The project includes a **dev service** (`internal/app/devservice/`) that emulates the upstream archive with fake content.
This should be used for all development and testing:

- Provides predictable, controlled test data
- Enables offline development without upstream dependencies
- Used by UI tests and integration tests

When working on this project, always use the dev service rather than the real upstream unless explicitly instructed otherwise.

## Architecture

```
cmd/erato/           → CLI entry point (Cobra)
internal/
├── app/             → Web frontend (Echo + Templ)
│   ├── component/   → Templ UI components
│   └── handler.go   → HTTP route handlers
├── archive/         → Core service (ConnectRPC)
│   ├── scraper.go   → HTML scraping from upstream
│   ├── hydrator.go  → Enriches data with user state
│   ├── interactivity.go → User interactions
│   ├── paginator.go → Cursor-based pagination
│   └── validator.go → Request/response validation
├── storage/         → SQLite persistence layer
│   └── db/          → SQLC queries + Goose migrations
├── content/         → Content transformation (HTML, Markdown)
├── sec/             → Authentication (HTTP Basic Auth)
├── command/         → CLI commands (serve, user)
└── config/          → YAML configuration via protobuf
proto/               → Protocol Buffer definitions
```

## Tech Stack

| Category       | Technology                  |
|----------------|-----------------------------|
| Language       | Go 1.25+                    |
| Web Framework  | Echo v4                     |
| RPC Framework  | ConnectRPC                  |
| Templating     | Templ                       |
| Database       | SQLite (modernc.org/sqlite) |
| SQL Generation | SQLC                        |
| Migrations     | Goose                       |
| Protobuf       | Buf CLI                     |
| Scraping       | GoQuery, Colly              |
| UI Testing     | Rod                         |
| CLI            | Cobra                       |

## CI/CD

GitHub Actions workflows in `.github/workflows/`:

- **ci.yaml** - Runs on PRs and merges to main/v2: generate, verify clean git, test, lint, proto breaking changes
- **ui.yaml** - Runs on PRs and merges to main/v2: UI tests
- **deps.yaml** - Weekly: updates Go deps, buf deps, htmx; opens PR

The shared setup action at `.github/actions/setup-go/` handles checkout, Go setup, and caching.

## Development Commands

Run `make help` to see all available targets.
Common commands:

```bash
make generate    # Run all code generators (protos, SQL, templ)
make build       # Compile binary to .tmp/erato
make watch       # Hot-reload development mode (air)
make format      # Format code (golangci-lint, buf, templ)
make lint        # Lint code and protos
make test        # Run all tests
make test-unit   # Run unit tests only
make test-ui     # Run UI tests only (Rod-based)
```

## Coding Conventions

### General

- Follow existing patterns in the codebase.
- Minimize external dependencies where possible.
- Use code generation (protos, SQLC, Templ) for type safety.
- Follow SOLID, YAGNI, and DRY principles, within reason.
- Tests and linting MUST be run before considering a set of changes as done.

### Logging

- The project exclusively uses `log/slog` for logging.
- Log messages should be string constant literals with no formatting.
- All variable information should be log attributes.
- When calling the variadic logging functions and methods, prefer the `slog.Attr` style over alternating strings and values.

### Error Handling

- All errors should be handled, with rare exception.
- Errors should either be logged at an appropriate error level if the user cannot reasonably respond to, or returned wrapped with context.
- Wrap errors with context using `fmt.Errorf("context: %w", err)` or a custom error if necessary.
- If the error is discarded, add a comment explaining why.
- Provide meaningful context about what operation failed.

### Testing

- Unit tests: `*_test.go` files alongside source.
- UI tests: `internal/uitest/` using Rod browser automation.
- Use `testify` for assertions.
- Use **table-driven tests** for comprehensive coverage where applicable.
- Strive for high unit test coverage of most library code.
- Do not jump through hoops to make mocks or fakes; prefer integration tests in these circumstances where appropriate.

Example pattern:

```go
tests := []struct {
    name    string
    input   Input
    want    Output
    wantErr bool
}{
    // test cases...
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

### Markdown

- Use one sentence per line for easier diff review.

### Tools

- Include documentation links at the top of config files.

### Configuration

Use the **functional options pattern** for configurable types:

```go
func WithOption(val Type) Option {
    return func(c *Config) {
        c.field = val
    }
}
```

## Key Entry Points

- **CLI Main**: `cmd/erato/main.go`
- **Serve Command**: `internal/command/serve.go` - starts RPC (`:9998` default) and web (`:9999` default) servers
- **Archive Service**: `internal/archive/archive.go` - ConnectRPC back-end service implementation
- **Web App**: `internal/app/app.go` - Web front-end Echo server implementation

## Service Layer Pattern

The archive service uses a decorator/chain pattern:

```
Scraper → Hydrator → Interactivity → Users → Paginator → Validator
```

Each layer wraps the previous and adds specific functionality.
The Validator is outermost to validate requests before processing and responses after.

## Configuration

Configuration is defined in Protocol Buffers (`proto/stolasapp/erato/v1/config.proto`) and loaded from YAML.
Key settings:

- `upstream.base_url` - Base URL of the archive to proxy
- `database.dsn` - SQLite connection string
- `rpc_server` / `web_server` - Server bind addresses

## Generated Code

Do not edit files in these locations directly:

- `internal/gen/` - Protocol Buffer generated code
- `internal/storage/db/*.sql.go` - SQLC generated code
- `internal/app/component/**/*_templ.go` - Templ generated code

Run `make generate` after modifying:

- `proto/**/*.proto` files
- `internal/storage/db/queries.sql`
- `internal/app/component/**/*.templ` files

## Protocol Buffers

### Style

This project follows the [AEP (API Enhancement Proposals)](https://aep.dev) style guide for Protocol Buffers.
The buf linter is configured to enforce AEP rules.
Key conventions:

- Resource messages use `path` field (field number 10018) as the identifier with `IDENTIFIER` behavior.
- Use AEP resource annotations with type, singular, plural, and pattern.
- Use field_behavior annotations: `REQUIRED`, `OUTPUT_ONLY`, `INPUT_ONLY`, `IMMUTABLE`, `OPTIONAL`, `IDENTIFIER`.
- Standard method patterns: `List*`, `Get*`, `Create*`, `Update*`, `Delete*`.
- Pagination uses `max_page_size`, `page_token`, `next_page_token`.
- Update methods use `google.protobuf.FieldMask` for partial updates.

### Protovalidate

Request validation uses [protovalidate](https://github.com/bufbuild/protovalidate) with rules defined in the proto files.
The `internal/archive/validator.go` decorator applies validation at runtime.

**Standard Rules**:

- `(buf.validate.field).required = true` - Field must be set
- `(buf.validate.field).string.min_len/max_len` - String length bounds
- `(buf.validate.field).int32.gte/lte` - Integer bounds
- `(buf.validate.field).cel` - Custom CEL expressions for complex validation

**Common Patterns**:

```protobuf
// Pagination bounds
int32 max_page_size = 2 [
  (buf.validate.field).int32 = {gte: 0, lte: 100}
];
string page_token = 3 [
  (buf.validate.field).string.max_len = 4096
];

// Required resource path
string path = 1 [
  (buf.validate.field).required = true
];

// Field mask validation (restrict to valid field names)
google.protobuf.FieldMask update_mask = 3 [
  (buf.validate.field).field_mask = {in: ["field1", "field2"]}
];

// Optional string format with empty allowed
string id = 1 [
  (buf.validate.field).cel = {
    id: "string.format_name"
    message: "must be 3-64 characters, alphanumeric and underscores only"
    expression: "this == '' || (this.size() >= 3 && this.size() <= 64 && this.matches('^[a-zA-Z0-9_]+$'))"
  }
];
```

**Key Learnings**:

- Use `this == '' || (validation)` pattern for optional fields that should only validate when set.
- Use `field_mask = {in: ["field1", "field2"]}` to restrict allowed field mask paths.
- Predefined rules with editions syntax are not well-supported; prefer inline CEL rules.
- The validation decorator should be the outermost in the chain to validate before/after all processing.
