# Style Guide

## Go Code

### Naming
- Use `camelCase` for unexported, `PascalCase` for exported
- Acronyms: `SSH`, `WG`, `IPsec`, `SS` — all caps or as defined in RFCs
- Tests: `TestFunctionName_Scenario`
- Benchmarks: `BenchmarkFunctionName`

### Imports
```go
import (
    // stdlib
    "fmt"
    "os"

    // external
    tea "github.com/charmbracelet/bubbletea"

    // internal
    "github.com/nyxora/nyxora/internal/config"
)
```

### Error Handling
- Always check errors. No `_ = fn()` without a comment.
- Wrap errors with context: `fmt.Errorf("connect: %w", err)`
- Use `errors.Is()` / `errors.As()` for sentinel errors

### Comments
- Exported functions MUST have doc comments
- TODO comments: `// TODO(handle): implement retry`
- No commented-out code in commits

### Concurrency
- Use `sync.Mutex` for shared state, not channels for simple locking
- Context propagation: first parameter in all blocking functions
- Prefer `errgroup` for parallel operations

## Git

### Commit Messages
```
<type>(<scope>): <description>

<body (optional)>

<footer (optional)>
```

Types: `feat`, `fix`, `refactor`, `docs`, `test`, `style`, `chore`, `perf`

### Branch Naming
- `feat/<name>` — new features
- `fix/<name>` — bug fixes  
- `refactor/<name>` — refactoring
- `docs/<name>` — documentation
- `chore/<name>` — maintenance

## Makefile
- Always quote variables: `$(BINARY)`
- Use `.PHONY` for non-file targets
- Prefer `$(MAKE)` over `make` in recursive calls
