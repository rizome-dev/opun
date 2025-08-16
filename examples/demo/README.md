# Cross-Provider SubAgent System Demo

This demo showcases Opun's revolutionary cross-provider subagent system that enables seamless coordination between Claude, Gemini, and Qwen AI providers.

## Overview

The Cross-Provider SubAgent System allows you to:
- **Leverage multiple AI providers** simultaneously for complex tasks
- **Route tasks intelligently** to the most capable provider
- **Execute workflows in parallel** across different providers
- **Chain outputs** between agents for sophisticated pipelines
- **Monitor performance** and optimize resource usage

## Prerequisites

1. **Install Opun:**
   ```bash
   cd /path/to/opun
   make install
   ```

2. **Optional: Install AI Providers**
   - Claude: `npm install -g @anthropic-ai/claude-code`
   - Gemini: Follow [Gemini CLI setup](https://github.com/google/gemini-cli)
   - Qwen: Install via `pip install qwen-code`

   Note: The demo will run in simulation mode if providers are not installed.

## Running the Demo

### Quick Start

```bash
# Run the comprehensive demo
./cross-provider-demo.sh
```

### What the Demo Does

The demo executes a complete code analysis and refactoring workflow:

1. **Creates Sample Code** - Generates a Go file with intentional inefficiencies
2. **Configures SubAgents** - Sets up specialized agents for each provider:
   - Claude: Code analysis specialist
   - Gemini: Refactoring expert
   - Qwen: Test generation specialist
3. **Executes Workflow** - Runs a multi-phase analysis:
   - Phase 1: Parallel code analysis
   - Phase 2: Refactoring based on analysis
   - Phase 3: Test generation
   - Phase 4: Final review and metrics
4. **Demonstrates Routing** - Shows intelligent task distribution
5. **Monitors Performance** - Tracks execution metrics

## Demo Output

The demo generates several artifacts:

### 1. Analysis Report (`analysis-report.md`)
A comprehensive report containing:
- Performance issues identified
- Refactored code
- Generated tests
- Improvement metrics

### 2. Performance Data (`performance.json`)
Detailed metrics including:
- Task execution times
- Success rates
- Routing decisions
- Provider utilization

### 3. Workflow Definition (`workflow.yaml`)
The complete workflow specification showing:
- Agent configurations
- Task dependencies
- Parallel execution setup
- Output chaining

## Key Features Demonstrated

### 1. Intelligent Task Routing

Tasks are automatically routed to the most suitable provider based on:
- **Capability matching** - Agent specializations
- **Performance history** - Past success rates
- **Priority scoring** - Task complexity assessment
- **Load balancing** - Even distribution of work

Example routing decision:
```json
{
  "task": "analyze-code",
  "routed_to": "claude-analyzer",
  "score": 95,
  "reason": "Best match for deep analysis capabilities"
}
```

### 2. Parallel Execution

Multiple agents work simultaneously on independent tasks:
```yaml
agents:
  - name: analyze-code
    subagent: claude-analyzer
    
  - name: analyze-structure
    subagent: gemini-refactor
    parallel: true  # Runs concurrently with above
```

### 3. Output Chaining

Results flow between agents using template variables:
```yaml
- name: refactor-code
  prompt: |
    Based on analysis: {{analyze-code.output}}
    Refactor the code to fix identified issues
  depends_on:
    - analyze-code
```

### 4. Provider Specialization

Each provider excels at different tasks:

| Provider | Specialization | Best For |
|----------|---------------|----------|
| Claude | Deep reasoning | Complex analysis, code review |
| Gemini | Code generation | Refactoring, optimization |
| Qwen | Testing | Test generation, validation |

## Advanced Usage

### Custom SubAgent Configuration

Create your own specialized agents:

```yaml
# ~/.opun/subagents/custom-agent.yaml
name: custom-analyzer
provider: claude
model: opus
capabilities:
  - security-analysis
  - vulnerability-detection
priority: 10
context:
  - "Focus on security best practices"
  - "Identify potential vulnerabilities"
```

### Complex Workflows

Build sophisticated multi-agent pipelines:

```yaml
# security-review.yaml
agents:
  - name: scan-vulnerabilities
    subagent: security-scanner
    
  - name: analyze-dependencies
    subagent: dependency-checker
    parallel: true
    
  - name: generate-report
    subagent: report-generator
    depends_on:
      - scan-vulnerabilities
      - analyze-dependencies
```

### Performance Optimization

Monitor and optimize agent performance:

```bash
# View agent statistics
opun subagent stats

# List active tasks
opun subagent tasks

# Benchmark providers
opun subagent benchmark
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Workflow Engine                 │
└────────────────────┬────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │     Task Router         │
        └────────────┬────────────┘
                     │
    ┌────────────────┼────────────────┐
    │                │                │
┌───▼───┐      ┌────▼────┐     ┌────▼────┐
│Claude │      │ Gemini  │     │  Qwen   │
│Agent  │      │  Agent  │     │  Agent  │
└───────┘      └─────────┘     └─────────┘
```

## Troubleshooting

### Providers Not Available
If you see "provider not available" messages, the demo runs in simulation mode. To use real providers:
1. Install the required provider CLI tools
2. Authenticate with your API keys
3. Re-run the demo

### Permission Issues
If you encounter permission errors:
```bash
chmod +x cross-provider-demo.sh
```

### Configuration Issues
Ensure Opun is properly configured:
```bash
opun setup check
```

## Next Steps

1. **Explore the generated workflow:**
   ```bash
   cat /tmp/opun-demo-*/workflow.yaml
   ```

2. **View the analysis report:**
   ```bash
   cat /tmp/opun-demo-*/analysis-report.md
   ```

3. **Create your own workflows:**
   - Copy and modify the demo workflow
   - Experiment with different agent configurations
   - Try different task combinations

4. **Integrate into your projects:**
   - Use subagents for code reviews
   - Automate testing workflows
   - Build CI/CD pipelines with AI assistance

## Learn More

- [SubAgent API Documentation](../../docs/subagent-api.md)
- [Workflow Configuration Guide](../../docs/workflows.md)
- [Provider Integration](../../docs/providers.md)
- [Performance Tuning](../../docs/performance.md)

## Support

For questions or issues:
- GitHub Issues: [github.com/rizome-dev/opun/issues](https://github.com/rizome-dev/opun/issues)
- Documentation: [opun.dev/docs](https://opun.dev/docs)
- Community: [discord.gg/opun](https://discord.gg/opun)

---

Built with ❤️ by Rizome Labs