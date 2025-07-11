# Contributing to Opun

We love your input! We want to make contributing to Opun as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## We Develop with GitHub
We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

## We Use [GitHub Flow](https://guides.github.com/introduction/flow/index.html)
Pull requests are the best way to propose changes to the codebase:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## Any contributions you make will be under the GPL-2.0 License
In short, when you submit code changes, your submissions are understood to be under the same [GPL-2.0 License](LICENSE) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report bugs using GitHub's [issues](https://github.com/rizome-dev/opun/issues)
We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/rizome-dev/opun/issues/new); it's that easy!

## Write bug reports with detail, background, and sample code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## Development Setup

1. Install Go 1.24 or later
2. Clone the repository:
   ```bash
   git clone https://github.com/rizome-dev/opun.git
   cd opun
   ```
3. Install dependencies:
   ```bash
   make deps
   ```
4. Run tests:
   ```bash
   make test
   ```
5. Build the binary:
   ```bash
   make build
   ```

## Code Style

- We use `gofmt` and `golangci-lint` for code formatting and linting
- Run `make fmt` to format your code
- Run `make lint` to check for linting issues
- Run `make check` to run all checks before submitting

## Testing

- Write tests for new functionality
- Ensure all tests pass with `make test`
- Check test coverage with `make test-coverage`
- Integration tests: `make test-integration`
- E2E tests: `make test-e2e`

## Documentation

- Update the README.md for any user-facing changes
- Add code comments for complex logic
- Update examples in the `examples/` directory when adding new features

## Pull Request Process

1. Update the README.md with details of changes to the interface
2. Update the examples with any new configuration options
3. The PR will be merged once you have the sign-off of maintainers

## Community

- Be respectful and inclusive
- Help others when you can
- Follow our [Code of Conduct](CODE_OF_CONDUCT.md)

## License
By contributing, you agree that your contributions will be licensed under its GPL-2.0 License.