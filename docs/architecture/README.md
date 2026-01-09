# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records documenting significant technical decisions made during the development of Open Model Gateway.

## What is an ADR?

An Architecture Decision Record captures an important architectural decision along with its context and consequences. ADRs help new team members understand why certain decisions were made and provide a historical record of the project's evolution.

## ADR Template

Each ADR follows this structure:

- **Status**: Proposed, Accepted, Deprecated, Superseded
- **Context**: The situation that led to this decision
- **Decision**: The change we're making
- **Consequences**: The resulting context after applying the decision

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [001](001-container-decomposition.md) | Container Decomposition | Accepted |
| [002](002-service-interfaces.md) | Service Interface Pattern | Accepted |
| [003](003-error-handling.md) | Standardized Error Handling | Accepted |

## Creating a New ADR

1. Copy the template structure from an existing ADR
2. Number sequentially (004, 005, etc.)
3. Use a descriptive filename: `NNN-brief-title.md`
4. Update this index
