# Contributing to FlowDeploy

Thank you for your interest in contributing to FlowDeploy! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.23+
- Node.js 20+
- pnpm 9+
- Docker Engine 24+
- Docker Compose v2
- PostgreSQL 16+

### Getting Started

1. **Clone the repository**

```bash
git clone https://github.com/your-org/paasdeploy.git
cd paasdeploy
```

2. **Install dependencies**

```bash
pnpm install
```

3. **Configure environment**

```bash
cp .env.example .env
# Edit .env with your local settings
```

4. **Start infrastructure**

```bash
pnpm docker:up
```

5. **Start development servers**

```bash
# Terminal 1: Backend
pnpm backend:dev

# Terminal 2: Frontend
pnpm dev
```

## Project Structure

```
paasdeploy/
├── apps/
│   ├── frontend/          # React application
│   │   ├── src/
│   │   │   ├── app/       # Bootstrap
│   │   │   ├── pages/     # Route components
│   │   │   ├── features/  # Domain logic
│   │   │   ├── components/# UI components
│   │   │   ├── services/  # API clients
│   │   │   ├── hooks/     # React hooks
│   │   │   └── types/     # TypeScript types
│   │   └── package.json
│   │
│   └── backend/           # Go application
│       ├── cmd/api/       # Entry point
│       ├── internal/
│       │   ├── config/    # Configuration
│       │   ├── domain/    # Domain entities
│       │   ├── handler/   # HTTP handlers
│       │   ├── service/   # Business logic
│       │   ├── repository/# Data access
│       │   └── engine/    # Deploy engine
│       └── go.mod
│
├── deploy/                # Infrastructure
├── docs/                  # Documentation
└── package.json           # Root workspace
```

## Coding Standards

### Go

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `go vet` before committing
- Write tests for new functionality
- Use dependency injection via structs (see `WorkerDeps` pattern)

#### Naming Conventions

```go
// Exported functions: PascalCase
func CreateApp(name string) error

// Private functions: camelCase
func validateAppName(name string) error

// Constants: PascalCase for exported, camelCase for private
const MaxRetries = 3
const defaultTimeout = 30 * time.Second
```

#### Error Handling

```go
// Always check errors
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Use domain errors for business logic
if !isValid {
    return domain.ErrInvalidInput
}
```

### TypeScript/React

- Use TypeScript strict mode
- Prefer functional components with hooks
- Use React Query for server state
- Follow the feature-based structure
- Use `??` instead of `||` for nullish coalescing
- Never use `any` type - always define proper types

### Linting and Formatting

The frontend uses ESLint and Prettier for code quality:

```bash
cd apps/frontend

# Check linting issues
pnpm lint

# Fix auto-fixable issues
pnpm lint:fix

# Format code
pnpm format

# Check formatting without changes
pnpm format:check
```

ESLint is configured to:

- Remove unused imports automatically
- Sort imports consistently
- Integrate with Prettier for formatting

#### Component Structure

```tsx
// components should be pure UI
interface ButtonProps {
  readonly variant: "primary" | "secondary";
  readonly onClick: () => void;
  readonly children: React.ReactNode;
}

export function Button({ variant, onClick, children }: ButtonProps) {
  return (
    <button className={cn(baseStyles, variants[variant])} onClick={onClick}>
      {children}
    </button>
  );
}
```

#### Hooks

```tsx
// Feature hooks encapsulate domain logic
export function useApps() {
  return useQuery({
    queryKey: ["apps"],
    queryFn: () => api.getApps(),
  });
}

export function useCreateApp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateAppInput) => api.createApp(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["apps"] });
    },
  });
}
```

### CSS/Styling

- Use Tailwind CSS utility classes
- Follow mobile-first approach
- Use CSS variables for theming
- Prefer composition over custom CSS

```tsx
// Good: Tailwind utilities
<div className="flex flex-col gap-4 p-4 md:flex-row md:gap-6">

// Avoid: Custom CSS unless necessary
<div style={{ display: 'flex', flexDirection: 'column' }}>
```

### Theme Support

When adding new components, ensure they support both light and dark themes:

```tsx
// Use semantic color variables
<div className="bg-background text-foreground">

// Use dark: variant for specific overrides
<div className="border-gray-200 dark:border-gray-700">
```

### Component Props

Mark component props as readonly (SonarQube S6759):

```tsx
interface ButtonProps {
  readonly variant: "primary" | "secondary";
  readonly onClick: () => void;
}
```

## VSCode Configuration

The project includes VSCode configurations for optimal developer experience:

### Debug Configurations

- **Frontend (Chrome/Edge)**: Launch browser with debugger attached
- **Backend (Go)**: Debug Go API with Delve
- **Full Stack**: Launch both frontend and backend simultaneously

Access via `Run and Debug` panel or `F5`.

### Tasks

Run common tasks via `Cmd/Ctrl + Shift + P` → `Tasks: Run Task`:

| Task             | Description                   |
| ---------------- | ----------------------------- |
| Frontend: Dev    | Start Vite dev server         |
| Frontend: Lint   | Run ESLint                    |
| Frontend: Format | Format with Prettier          |
| Backend: Dev     | Start Go API with hot reload  |
| Backend: Build   | Build Go binary               |
| Docker: Up       | Start infrastructure services |
| Docker: Down     | Stop infrastructure services  |

### Recommended Extensions

Install recommended extensions via `Extensions` → `Show Recommended Extensions`.

## Git Workflow

### Branch Naming

```
feature/add-webhook-support
fix/deploy-queue-race-condition
docs/update-architecture
refactor/extract-executor
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add rollback functionality
fix: prevent race condition in deploy queue
docs: update API documentation
refactor: extract command executor
test: add unit tests for health checker
chore: update dependencies
```

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear commits
3. Ensure tests pass
4. Update documentation if needed
5. Create a pull request with description
6. Wait for code review

### PR Description Template

```markdown
## Summary

Brief description of the changes.

## Changes

- Added X functionality
- Fixed Y bug
- Updated Z documentation

## Testing

- [ ] Unit tests added/updated
- [ ] Manual testing performed
- [ ] All tests passing

## Screenshots (if applicable)
```

## Testing

### Backend Tests

```bash
cd apps/backend

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/engine/...
```

### Frontend Tests

```bash
cd apps/frontend

# Run tests
pnpm test

# Run with coverage
pnpm test:coverage

# Run in watch mode
pnpm test:watch
```

### Writing Tests

#### Go Tests

```go
func TestWorker_ExecuteDeploy(t *testing.T) {
    // Arrange
    worker := NewWorker(mockExecutor, mockNotifier)
    deploy := &domain.Deployment{
        ID:        "test-id",
        AppID:     "app-id",
        CommitSHA: "abc123",
    }

    // Act
    err := worker.Execute(context.Background(), deploy)

    // Assert
    if err != nil {
        t.Errorf("expected no error, got %v", err)
    }
}
```

#### React Tests

```tsx
describe("AppCard", () => {
  it("should render app name and status", () => {
    const app = {
      id: "1",
      name: "my-app",
      status: "deployed",
    };

    render(<AppCard app={app} />);

    expect(screen.getByText("my-app")).toBeInTheDocument();
    expect(screen.getByText("deployed")).toBeInTheDocument();
  });
});
```

## Code Review Guidelines

### For Authors

- Keep PRs focused and small
- Provide context in the description
- Respond to feedback constructively
- Update based on review comments

### For Reviewers

- Be constructive and respectful
- Explain the "why" behind suggestions
- Approve when ready, don't block unnecessarily
- Focus on:
  - Correctness
  - Security
  - Performance
  - Readability
  - Test coverage

## Security Guidelines

### Do NOT

- Execute shell commands with `sh -c`
- Store secrets in code or frontend bundle
- Run processes as root
- Trust user input without validation
- Commit `.env` files

### DO

- Use explicit command arguments with `os/exec`
- Validate and sanitize all inputs
- Use environment variables for secrets
- Apply principle of least privilege
- Review dependencies for vulnerabilities

## Documentation

### Code Comments

```go
// Executor runs OS commands safely without shell interpretation.
// It uses os/exec directly with explicit arguments to prevent
// shell injection attacks.
type Executor struct {
    workDir string
    timeout time.Duration
}
```

### API Documentation

Document all public API endpoints with:

- HTTP method and path
- Request/response format
- Error codes
- Example usage

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions for questions
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
