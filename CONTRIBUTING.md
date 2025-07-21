# Contributing to Railway PostgreSQL Backup

Thank you for your interest in contributing to Railway PostgreSQL Backup! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## How to Contribute

### Reporting Issues

1. Check existing issues to avoid duplicates
2. Use the issue template when available
3. Provide clear reproduction steps
4. Include relevant environment details

### Suggesting Features

1. Open an issue with the "feature request" label
2. Clearly describe the use case
3. Explain why existing functionality doesn't meet your needs
4. Be open to discussion and alternative approaches

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes following our guidelines
4. Submit a pull request with a clear description

## Development Guidelines

### Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/railway-postgres-backup
cd railway-postgres-backup

# Add upstream remote
git remote add upstream https://github.com/imedwei/railway-postgres-backup

# Install dependencies
go mod download

# Run tests
task test
```

### Code Style

- Use `gofmt` for formatting (automatically done by `task fmt`)
- Follow Go idioms and best practices
- Write clear, self-documenting code
- Add comments for complex logic

### Testing

- Write unit tests for all new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Test both success and error cases

### Commit Messages

Follow conventional commit format:

```
type(scope): subject

body

footer
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or fixes
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

Example:
```
feat(storage): add Azure Blob Storage support

- Implement Azure storage interface
- Add configuration for Azure credentials
- Include comprehensive unit tests

Closes #123
```

### Pull Request Process

1. Update documentation for any API changes
2. Add tests for new functionality
3. Ensure all tests pass (`task test`)
4. Format code (`task fmt`)
5. Update README.md if needed
6. Request review from maintainers

### Review Criteria

PRs are evaluated on:
- Code quality and clarity
- Test coverage
- Documentation updates
- Adherence to project standards
- Performance impact
- Security considerations

## Development Workflow

### Interface-First Design

When adding new functionality:
1. Define interfaces first
2. Write tests against interfaces
3. Implement concrete types
4. Ensure proper error handling

### Adding Storage Backends

1. Implement the `Storage` interface
2. Add configuration in `internal/config/`
3. Update factory in `internal/storage/factory.go`
4. Write comprehensive tests
5. Update documentation

### Metrics and Monitoring

- Add Prometheus metrics for observable behavior
- Use consistent metric naming
- Document new metrics in README

## Testing

### Running Tests

```bash
# All tests
task test

# With coverage
task test:coverage

# Specific package
go test ./internal/storage/...
```

### Writing Tests

- Use meaningful test names
- Test edge cases
- Mock external dependencies
- Verify error conditions
- Check metric recording

## Documentation

- Update README for user-facing changes
- Add godoc comments for exported types
- Include examples where helpful
- Keep CLAUDE.md updated for AI assistance

## Release Process

1. Ensure all tests pass
2. Update version numbers
3. Update CHANGELOG.md
4. Tag release following semver
5. GitHub Actions handles Docker builds

## Getting Help

- Open an issue for questions
- Join discussions in pull requests
- Check existing documentation
- Review similar implementations

## Recognition

Contributors are recognized in:
- Git commit history
- GitHub contributors page
- Release notes for significant contributions

Thank you for contributing to Railway PostgreSQL Backup!