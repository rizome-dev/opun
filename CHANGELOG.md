# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Opun TDK (Terminal Development Kit)
- AI provider abstraction for Claude Code, Gemini CLI, and Qwen Code
- Cross-provider subagent orchestration system
- Intelligent task routing and delegation based on capabilities
- YAML-based workflow orchestration system with subagent support
- Plugin system with Go, shell script, and JSON support
- PromptGarden for centralized prompt management
- MCP (Model Context Protocol) integration with Task server support
- Session management with isolation and persistence
- Interactive chat mode with slash commands
- PTY automation for terminal control
- Multi-agent coordination with dependencies
- Parallel and sequential subagent execution
- Retry logic with exponential backoff
- Variable substitution and file inclusion in workflows
- Homebrew distribution support
- Comprehensive CI/CD pipeline
- Subagent CLI commands (list, create, delete, execute, info)

### Security
- Secure session data storage in user home directory
- Provider authentication handling
- Safe PTY session management

[Unreleased]: https://github.com/rizome-dev/opun/compare/v0.1.0...HEAD