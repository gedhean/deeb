# Contributing to Deeb

Thank you for your interest in contributing to Deeb! This guide will help you get started with the development workflow and contribution process.

## Quick Links

- [Setup Guide](docs/setup.md) - Complete setup instructions
- [Development Guide](docs/dev-guide.md) - Docker development environment
- [Architecture Overview](docs/architecture.png) - System architecture
- [Project Documentation](docs/project.md) - Technical design and decisions

## Getting Started

### Prerequisites

- Go 1.25.3+
- Node.js 24+ with pnpm
- PostgreSQL 16+
- Docker & Docker Compose (recommended)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/NSXBet/deeb.git
cd deeb

# Start development environment (Frontend and Backend will start)
make dev
```

See the [Setup Guide](docs/SETUP.md) for detailed instructions.

## Development Workflow

### Design Principles

We follow these core principles:

- **KISS (Keep It Simple)**: Write straightforward, uncomplicated solutions
- **YAGNI (You Aren't Gonna Need It)**: Implement only what's currently needed
- **SOLID Principles**: Follow single responsibility, open-closed, and other SOLID principles

### Backend Development

**After making Go code changes:**

```bash
# Run tests
make test                   # Unit tests
make test-integration      # Integration tests
make test-coverage         # Coverage report

# Check code quality
make lint                  # Run linters
make lint-fix              # Auto-fix issues
make format                # Format code

# Start development server with Air (live reload)
make run-air               # With live reload
```

**Example workflow:**

```bash
cd backend

# Make your changes
vim internal/handlers/changeplans.go

# Run relevant tests
go test ./internal/handlers/...

# Format and lint
make format
make lint-fix
make lint

# Verify everything works
make test
```

### Frontend Development

**After making frontend code changes:**

```bash
# Run tests
pnpm --dir frontend run test

# Check code quality
pnpm --dir frontend run lint
pnpm --dir frontend run lint-fix
pnpm --dir frontend run format

# Start development server
pnpm --dir frontend run dev
```

**Example workflow:**

```bash
# Make your changes
vim frontend/src/components/ChangePlanForm.tsx

# Run tests
pnpm --dir frontend run test

# Format and lint
pnpm --dir frontend run format
pnpm --dir frontend run lint-fix
pnpm --dir frontend run lint

# Start dev server
pnpm --dir frontend run dev
```

### Running the Full Stack

```bash
make dev
```

**Access:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- API Docs: http://localhost:8080/api/docs

### Make Changes

- Write clean, readable code following project conventions
- Add tests for new functionality
- Update documentation as needed
- Follow the design principles (KISS, YAGNI, SOLID)

### Test Your Changes

**Backend:**
```bash
make -C backend test
make -C backend lint-fix
make -C backend format
```

**Frontend:**
```bash
pnpm --dir frontend run test
pnpm --dir frontend run lint-fix
pnpm --dir frontend run format
```

### 4. Commit Your Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "feat: add support for PostgreSQL connection pooling"
```

**Commit message format:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `test:` - Test changes
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

### 5. Push and Create Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name

# Create a pull request on GitHub
```

**Pull request checklist:**
- [ ] Tests pass (`make test` and `pnpm --dir frontend run test`)
- [ ] Code is formatted (`make format` and `pnpm --dir frontend run format`)
- [ ] Linting passes (`make lint` and `pnpm --dir frontend run lint`)
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] PR description explains the changes

## Code Standards

### Go Backend

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Write table-driven tests
- Keep functions small and focused
- Document exported functions and types

**Example:**

```go
// GetChangePlanByID retrieves a change plan by its ID.
// Returns ErrNotFound if the change plan doesn't exist.
func (s *Service) GetChangePlanByID(ctx context.Context, id string) (*ChangePlan, error) {
    // Implementation
}
```

### React Frontend

- Use functional components with hooks
- Follow TypeScript strict mode
- Use shadcn/ui components consistently
- Keep components focused and reusable
- Write tests for complex logic

**Example:**

```typescript
interface ChangePlanFormProps {
  onSubmit: (data: ChangePlanData) => void;
  initialData?: ChangePlanData;
}

export function ChangePlanForm({ onSubmit, initialData }: ChangePlanFormProps) {
  // Implementation
}
```

## Testing

### Backend Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/handlers/...

# Run with coverage
make test-coverage

# Run integration tests
make test-integration
```

### Frontend Tests

```bash
# Run all tests
pnpm --dir frontend run test

# Run in watch mode
pnpm --dir frontend run test:watch

# Run with coverage
pnpm --dir frontend run test:coverage
```

## Common Tasks

### Adding a New Database Connection Type

1. Update `internal/database/connector.go`
2. Add connection logic in `internal/database/`
3. Update tests in `internal/database/*_test.go`
4. Update documentation

### Adding a New API Endpoint

1. Define route in `cmd/server/main.go`
2. Create handler in `internal/handlers/`
3. Add service logic in `internal/services/`
4. Update OpenAPI spec in `api/openapi.yaml`
5. Add tests

### Adding a New UI Component

1. Create component in `frontend/src/components/`
2. Add TypeScript types
3. Use shadcn/ui primitives
4. Add component tests
5. Update Storybook (if applicable)

## Troubleshooting

### Backend Issues

**Database connection errors:**
```bash
# Check PostgreSQL is running
docker compose ps postgres

# Check connection
docker compose exec postgres pg_isready -U deeb -d deeb
```

**Build errors:**
```bash
# Clean and rebuild
make clean
go mod tidy
go build ./...
```

### Frontend Issues

**Dependency issues:**
```bash
# Clean install
rm -rf frontend/node_modules
pnpm --dir frontend install
```

**Build errors:**
```bash
# Clear cache
rm -rf frontend/.vite
pnpm --dir frontend run build
```

## Getting Help

- Check the [documentation](docs/)
- Review existing [issues](https://github.com/NSXBet/deeb/issues)
- Create a new issue with detailed information
- Join our community discussions

## Code Review Process

1. **Automated checks**: CI runs tests and linting
2. **Peer review**: At least one approval required
3. **Architecture review**: For significant changes
4. **Security review**: For security-related changes

## License

By contributing to Deeb, you agree that your contributions will be licensed under the project's license.

---

Thank you for contributing to Deeb!
