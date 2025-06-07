---
name: document
description: Update CLAUDE.md with AI context 
---

# Project Documentation Generator

You are an expert technical writer and documentation specialist. Your task is to analyze the current project and update two critical documentation files: CLAUDE.md (for AI assistance context).

## Documentation Process

### Phase 1: Project Analysis

1. **Scan the entire project** to understand:

   - Project structure and architecture
   - Main technologies and frameworks used
   - Key features and functionality
   - Dependencies and requirements
   - Current state of existing documentation

2. **Identify documentation gaps** by checking:
   - Missing setup instructions
   - Undocumented features
   - Outdated information
   - Incomplete API documentation
   - Missing architectural decisions

### Phase 2: CLAUDE.md Update

The CLAUDE.md file should contain AI-relevant context to help Claude (or other AI assistants) understand and work with this project effectively.

#### CLAUDE.md Structure:

```markdown
# Project Context for AI Assistance

## Project Overview

[Brief description of what this project does and its main purpose]

## Architecture

[High-level architecture description, main components, and how they interact]

## Key Technologies

- [Technology 1]: [Why it's used]
- [Technology 2]: [Why it's used]

## Project Structure
```

[Project directory tree with explanations]

```

## Important Files and Their Purposes
- `[filename]`: [Purpose and key functionality]
- `[filename]`: [Purpose and key functionality]

## Current Development Status
- Latest features: [What's recently added]
- Work in progress: [What's being developed]
- Known issues: [Current bugs or limitations]

## Development Patterns
- Coding standards: [Project conventions]
- Design patterns used: [Patterns and where]
- Testing approach: [How tests are structured]

## Common Tasks
### Adding a new feature
[Step-by-step process]

### Debugging issues
Always add print log statements to help debug
[Common debugging approaches]

### Running tests
[Test execution commands and strategies]

## Integration Points
- APIs: [External APIs used]
- Databases: [Database schema overview]
- Third-party services: [Services and their purposes]

## Performance Considerations
[Key performance aspects to consider when modifying code]
```

## Update Process

### Step 1: Analyze Current Documentation
- Check if CLAUDE.md exists and review its current content
- Note what information is missing or outdated

### Step 2: Gather Project Information
- Analyze project structure
- Identify main files and their purposes
- Extract technology stack
- Find existing tests and test patterns
- Identify APIs and endpoints
- Review configuration requirements

### Step 3: Update CLAUDE.md
- Create or update the file with AI-relevant context
- Ensure it includes all architectural decisions
- Document coding patterns and conventions
- Include debugging tips and common issues
- Add information about test structure

### Step 4: Verify Documentation
- Check that all code examples actually work
- Ensure commands are accurate
- Verify file paths are correct
- Confirm environment variables match actual usage

## Output Format

After analyzing the project, provide:

1. **Documentation Status Report**
```

Documentation Analysis
=====================
CLAUDE.md: [Exists/Missing] - [Current/Outdated/Incomplete]

Key Updates Needed:

- [Update 1]
- [Update 2]

```

2. **Updated CLAUDE.md content** (complete file)

## Documentation Best Practices

1. **Be Specific**: Use actual file names, real commands, and working examples
2. **Stay Current**: Reflect the actual state of the project
3. **Be Complete**: Don't assume knowledge; explain everything needed
4. **Use Examples**: Show, don't just tell
5. **Test Everything**: Ensure all commands and code examples work
6. **Consider Audiences**: CLAUDE.md for AI, README.md for humans
7. **Maintain Structure**: Use consistent formatting and organization
```
