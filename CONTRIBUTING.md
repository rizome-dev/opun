# Contributing to Opun

## Code Contributions

Pull requests are the best way to propose changes to the codebase. Before you do so, run through this checklist:

1. If you've added code that should be tested, add tests.
2. If you've added a new feature with a distinct configuration, a sample should be added to `examples` for easy maintainer testing.
3. Run `make test` & `make build` to ensure they pass.
4. Wait for CI tests to pass (for now, Lint & Security are not required, but we will be reviewing the pipeline to ensure you haven't added any HIGH vuln security issues - if so, your PR will be rejected, so I highly reccomend carefully reading the results of the Security CI step).

### Development Environment Basics

Setup a new environment:
   ```bash
   make setup
   ```
Run tests:
   ```bash
   make test
   make test-coverage
   make test-integration
   make test-e2e
   ```
Build the binary:
   ```bash
   make build
   ```
Format your code:
   ```bash
   make fmt
   ```
Check for linting issues:
   ```bash
   make lint
   ```
Run all checks before a push:
   ```bash
   make check
   ```

## Other Contributions

### Report bugs using GitHub's [issues](https://github.com/rizome-dev/opun/issues)

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/rizome-dev/opun/issues/new).

### Write bug reports with detail, background, and sample code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## License

By contributing, you agree that your contributions will be licensed under its GPL-2.0 License.
