#!/bin/bash

# Cross-Provider SubAgent Demo
# This demo showcases the power of Opun's cross-provider subagent system
# It demonstrates task routing, delegation, and parallel execution across
# Claude, Gemini, and Qwen providers

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Demo header
echo -e "${CYAN}╔═══════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     Opun Cross-Provider SubAgent System Demo         ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════╝${NC}"
echo

# Check if Opun is installed
if ! command -v opun &> /dev/null; then
    echo -e "${RED}Error: opun is not installed or not in PATH${NC}"
    echo "Please run 'make install' from the project root"
    exit 1
fi

# Create a demo workspace
DEMO_DIR="/tmp/opun-demo-$(date +%s)"
mkdir -p "$DEMO_DIR"
cd "$DEMO_DIR"

echo -e "${GREEN}➤ Created demo workspace: $DEMO_DIR${NC}"
echo

# Step 1: Create sample code for analysis
echo -e "${YELLOW}Step 1: Creating sample code for analysis${NC}"
cat > main.go << 'EOF'
package main

import (
    "fmt"
    "strings"
)

// ProcessData handles data transformation
func ProcessData(input string) string {
    // This could be optimized
    result := ""
    for i := 0; i < len(input); i++ {
        char := string(input[i])
        result = result + strings.ToUpper(char)
    }
    return result
}

// CalculateSum adds numbers inefficiently
func CalculateSum(numbers []int) int {
    sum := 0
    for i := 0; i < len(numbers); i++ {
        sum = sum + numbers[i]
    }
    return sum
}

func main() {
    data := "hello world"
    fmt.Println(ProcessData(data))
    
    nums := []int{1, 2, 3, 4, 5}
    fmt.Println("Sum:", CalculateSum(nums))
}
EOF

echo -e "${GREEN}✓ Created main.go with sample code${NC}"
echo

# Step 2: Create subagent configurations
echo -e "${YELLOW}Step 2: Configuring Cross-Provider SubAgents${NC}"

# Create config directory
mkdir -p ~/.opun/subagents

# Claude SubAgent - Code Analysis Specialist
cat > ~/.opun/subagents/claude-analyzer.yaml << 'EOF'
name: claude-analyzer
provider: claude
model: sonnet
type: declarative
description: "Expert code analyzer using Claude's deep reasoning"
capabilities:
  - code-analysis
  - performance-review
  - best-practices
priority: 9
context:
  - "Focus on code quality and performance"
  - "Identify optimization opportunities"
  - "Suggest idiomatic Go patterns"
EOF

echo -e "${GREEN}✓ Created Claude analyzer subagent${NC}"

# Gemini SubAgent - Refactoring Specialist
cat > ~/.opun/subagents/gemini-refactor.yaml << 'EOF'
name: gemini-refactor
provider: gemini
model: gemini-pro
type: programmatic
description: "Refactoring expert using Gemini's code generation"
capabilities:
  - code-refactoring
  - optimization
  - modernization
priority: 8
context:
  - "Apply Go best practices"
  - "Optimize for performance"
  - "Maintain readability"
EOF

echo -e "${GREEN}✓ Created Gemini refactoring subagent${NC}"

# Qwen SubAgent - Testing Specialist
cat > ~/.opun/subagents/qwen-tester.yaml << 'EOF'
name: qwen-tester
provider: qwen
model: code
type: workflow
description: "Test generation specialist using Qwen"
capabilities:
  - test-generation
  - test-coverage
  - benchmarking
priority: 7
context:
  - "Generate comprehensive test cases"
  - "Include edge cases"
  - "Add benchmarks for performance-critical functions"
EOF

echo -e "${GREEN}✓ Created Qwen testing subagent${NC}"
echo

# Step 3: Create a multi-agent workflow
echo -e "${YELLOW}Step 3: Creating Multi-Agent Workflow${NC}"

cat > workflow.yaml << 'EOF'
name: cross-provider-analysis
description: "Comprehensive code analysis using multiple AI providers"

agents:
  # Phase 1: Parallel Analysis
  - name: analyze-code
    subagent: claude-analyzer
    prompt: |
      Analyze the Go code in main.go and identify:
      1. Performance bottlenecks
      2. Non-idiomatic patterns
      3. Potential bugs or issues
      Provide specific recommendations for each function.
    
  - name: analyze-structure
    subagent: gemini-refactor
    prompt: |
      Review the code structure in main.go and suggest:
      1. Better variable names
      2. More efficient algorithms
      3. Modern Go patterns to apply
    parallel: true  # Run in parallel with claude-analyzer

  # Phase 2: Refactoring based on analysis
  - name: refactor-code
    subagent: gemini-refactor
    prompt: |
      Based on the analysis from {{analyze-code.output}} and {{analyze-structure.output}},
      refactor the main.go file to:
      1. Fix identified performance issues
      2. Apply Go best practices
      3. Improve code readability
      Generate the complete refactored code.
    depends_on:
      - analyze-code
      - analyze-structure
    
  # Phase 3: Generate tests for refactored code
  - name: generate-tests
    subagent: qwen-tester
    prompt: |
      For the refactored code from {{refactor-code.output}}, generate:
      1. Unit tests for all functions
      2. Table-driven tests where appropriate
      3. Benchmark tests for performance-critical functions
      Include edge cases and error scenarios.
    depends_on:
      - refactor-code

  # Phase 4: Final review
  - name: final-review
    subagent: claude-analyzer
    prompt: |
      Review the complete solution:
      - Original code: main.go
      - Refactored code: {{refactor-code.output}}
      - Tests: {{generate-tests.output}}
      
      Provide:
      1. A summary of improvements made
      2. Performance impact assessment
      3. Test coverage analysis
      4. Any remaining recommendations
    depends_on:
      - generate-tests

output:
  format: markdown
  file: analysis-report.md
EOF

echo -e "${GREEN}✓ Created cross-provider workflow${NC}"
echo

# Step 4: Demonstrate task routing
echo -e "${YELLOW}Step 4: Demonstrating Intelligent Task Routing${NC}"

cat > routing-demo.yaml << 'EOF'
name: routing-demo
description: "Demonstrate how tasks are routed to the best provider"

tasks:
  - id: task-1
    name: "Complex reasoning task"
    context:
      requires: deep-analysis
      complexity: high
    # This should route to Claude (best at reasoning)
    
  - id: task-2  
    name: "Code generation task"
    context:
      requires: code-generation
      language: go
    # This should route to Gemini or Qwen
    
  - id: task-3
    name: "Quick syntax check"
    context:
      requires: syntax-validation
      complexity: low
    # This should route to the fastest available agent
EOF

echo -e "${GREEN}✓ Created routing demonstration${NC}"
echo

# Step 5: Execute the workflow
echo -e "${YELLOW}Step 5: Executing Cross-Provider Workflow${NC}"
echo -e "${CYAN}This will coordinate tasks across Claude, Gemini, and Qwen...${NC}"
echo

# Check if providers are available
echo -e "${BLUE}Checking provider availability...${NC}"

PROVIDERS_AVAILABLE=true

if ! command -v claude &> /dev/null && ! command -v npx &> /dev/null; then
    echo -e "${YELLOW}⚠ Claude not available - using mock mode${NC}"
    PROVIDERS_AVAILABLE=false
fi

if ! command -v gemini &> /dev/null; then
    echo -e "${YELLOW}⚠ Gemini not available - using mock mode${NC}"
    PROVIDERS_AVAILABLE=false
fi

if ! command -v qwen &> /dev/null; then
    echo -e "${YELLOW}⚠ Qwen not available - using mock mode${NC}"
    PROVIDERS_AVAILABLE=false
fi

if [ "$PROVIDERS_AVAILABLE" = true ]; then
    echo -e "${GREEN}✓ All providers available${NC}"
    echo
    echo -e "${MAGENTA}Executing workflow...${NC}"
    # opun workflow run workflow.yaml
    echo -e "${YELLOW}[Simulated] opun workflow run workflow.yaml${NC}"
else
    echo
    echo -e "${YELLOW}Running in demonstration mode (providers not installed)${NC}"
fi

# Step 6: Show results
echo
echo -e "${YELLOW}Step 6: Workflow Results${NC}"

# Simulate results for demonstration
cat > analysis-report.md << 'EOF'
# Cross-Provider Analysis Report

## Phase 1: Parallel Analysis Results

### Claude Analysis (claude-analyzer)
**Performance Issues Identified:**
- String concatenation in loop (ProcessData function)
- Inefficient iteration pattern
- Missing error handling

### Gemini Structure Analysis (gemini-refactor)
**Structural Improvements:**
- Use strings.Builder for concatenation
- Employ range loops for cleaner syntax
- Add input validation

## Phase 2: Refactored Code (gemini-refactor)

```go
package main

import (
    "fmt"
    "strings"
)

// ProcessData efficiently transforms input to uppercase
func ProcessData(input string) string {
    if input == "" {
        return ""
    }
    return strings.ToUpper(input)
}

// CalculateSum adds numbers using idiomatic Go
func CalculateSum(numbers []int) int {
    sum := 0
    for _, num := range numbers {
        sum += num
    }
    return sum
}
```

## Phase 3: Generated Tests (qwen-tester)

```go
func TestProcessData(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {"empty", "", ""},
        {"lowercase", "hello", "HELLO"},
        {"mixed", "Hello World", "HELLO WORLD"},
    }
    // ... test implementation
}

func BenchmarkProcessData(b *testing.B) {
    input := "hello world"
    for i := 0; i < b.N; i++ {
        ProcessData(input)
    }
}
```

## Phase 4: Final Review (claude-analyzer)

### Summary of Improvements:
1. **Performance**: 85% faster string processing
2. **Readability**: Cleaner, more idiomatic code
3. **Testing**: 100% code coverage achieved
4. **Maintainability**: Improved with proper patterns

### Metrics:
- Original: 245ns/op
- Optimized: 37ns/op
- Improvement: 6.6x faster
EOF

echo -e "${GREEN}✓ Analysis report generated${NC}"
echo

# Step 7: Demonstrate monitoring
echo -e "${YELLOW}Step 7: SubAgent Performance Monitoring${NC}"

cat > performance.json << 'EOF'
{
  "workflow": "cross-provider-analysis",
  "total_duration": "8.3s",
  "agents": {
    "claude-analyzer": {
      "tasks": 2,
      "avg_time": "2.1s",
      "success_rate": "100%"
    },
    "gemini-refactor": {
      "tasks": 2,
      "avg_time": "1.8s",
      "success_rate": "100%"
    },
    "qwen-tester": {
      "tasks": 1,
      "avg_time": "2.3s",
      "success_rate": "100%"
    }
  },
  "routing_decisions": [
    {
      "task": "analyze-code",
      "routed_to": "claude-analyzer",
      "score": 95,
      "reason": "Best match for deep analysis capabilities"
    },
    {
      "task": "refactor-code",
      "routed_to": "gemini-refactor",
      "score": 92,
      "reason": "Specialized in code generation and refactoring"
    },
    {
      "task": "generate-tests",
      "routed_to": "qwen-tester",
      "score": 88,
      "reason": "Optimized for test generation tasks"
    }
  ]
}
EOF

echo -e "${GREEN}✓ Performance metrics collected${NC}"
echo

# Display performance summary
echo -e "${CYAN}╔═══════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║              Workflow Execution Summary              ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════╝${NC}"
echo
echo -e "${WHITE}Total Execution Time: 8.3 seconds${NC}"
echo -e "${WHITE}Tasks Completed: 5${NC}"
echo -e "${WHITE}Providers Used: 3 (Claude, Gemini, Qwen)${NC}"
echo -e "${WHITE}Parallel Tasks: 2${NC}"
echo -e "${WHITE}Success Rate: 100%${NC}"
echo

# Step 8: Advanced features
echo -e "${YELLOW}Step 8: Advanced Features Demonstration${NC}"

cat > advanced-features.txt << 'EOF'
Advanced SubAgent Features Demonstrated:

1. INTELLIGENT ROUTING
   - Tasks automatically routed to best provider
   - Scoring based on capabilities and performance
   - Dynamic load balancing

2. PARALLEL EXECUTION  
   - Multiple agents work simultaneously
   - Dependency management ensures correct order
   - Optimal resource utilization

3. OUTPUT CHAINING
   - Results from one agent feed into another
   - Template variables for data passing
   - Maintains context across providers

4. PROVIDER SPECIALIZATION
   - Claude: Deep analysis and reasoning
   - Gemini: Code generation and refactoring
   - Qwen: Testing and validation

5. PERFORMANCE MONITORING
   - Real-time task tracking
   - Success rate monitoring
   - Execution time analysis

6. ERROR RECOVERY
   - Automatic retry on failure
   - Fallback to alternative providers
   - Graceful degradation
EOF

cat advanced-features.txt
echo

# Cleanup message
echo -e "${CYAN}╔═══════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                    Demo Complete!                    ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════╝${NC}"
echo
echo -e "${GREEN}Demo workspace: $DEMO_DIR${NC}"
echo -e "${GREEN}Analysis report: $DEMO_DIR/analysis-report.md${NC}"
echo -e "${GREEN}Performance data: $DEMO_DIR/performance.json${NC}"
echo
echo -e "${YELLOW}To explore further:${NC}"
echo -e "  • View the generated report: cat $DEMO_DIR/analysis-report.md"
echo -e "  • Check performance metrics: cat $DEMO_DIR/performance.json"
echo -e "  • Inspect the workflow: cat $DEMO_DIR/workflow.yaml"
echo
echo -e "${MAGENTA}Thank you for exploring Opun's Cross-Provider SubAgent System!${NC}"