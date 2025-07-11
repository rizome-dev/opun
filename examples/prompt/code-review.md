---
name: code-review
description: Comprehensive code review with customizable focus areas
category: development
tags: 
  - review
  - quality
  - best-practices
version: 1.0.0
variables:
  - name: file_path
    description: Path to the file to review
    required: true
  - name: language
    description: Programming language
    required: false
    default: "auto-detect"
  - name: focus_areas
    description: Specific areas to focus on
    required: false
    default: "all"
---

# Code Review Request

Please perform a comprehensive code review of the following file:

**File**: `{{file_path}}`  
**Language**: {{language}}

## Review Scope

{{#if focus_areas}}
Focus particularly on: {{focus_areas}}
{{else}}
Please review all aspects of the code.
{{/if}}

## Review Checklist

1. **Code Quality**
   - Is the code readable and well-organized?
   - Are variable and function names descriptive?
   - Is the code DRY (Don't Repeat Yourself)?

2. **Best Practices**
   - Does it follow {{language}} conventions?
   - Are there any anti-patterns?
   - Is error handling appropriate?

3. **Performance**
   - Are there any obvious performance issues?
   - Could any algorithms be optimized?
   - Are there unnecessary computations?

4. **Security**
   - Are there any security vulnerabilities?
   - Is input validation adequate?
   - Are secrets handled properly?

5. **Testing**
   - Is the code testable?
   - What test cases would you recommend?
   - Are edge cases handled?

## Output Format

Please provide:
1. An overall assessment (1-10 rating)
2. Specific issues found (categorized by severity)
3. Suggested improvements with code examples
4. Positive aspects worth preserving

Be constructive and educational in your feedback.