# ADR 001: Container Decomposition

## Status
Accepted

## Context
The original `app.Container` struct was a "god object" containing all application dependencies, making it difficult to understand component responsibilities and test individual subsystems.

## Decision
Decompose the Container into focused sub-containers:

1. **ServiceContainer** - Business logic services (tenant, usage, billing)
2. **RoutingContainer** - Provider factory, routing engine, health checks
3. **DataContainer** - Database pool, Redis client, SQLC queries
4. **TelemetryContainer** - OTEL tracing, Prometheus metrics
5. **RateLimitContainer** - Rate limiting state and enforcement

The main `Container` composes these sub-containers and provides convenience accessors.

## Structure
```
internal/app/
├── container.go         # Main Container composing sub-containers
└── containers/
    ├── service.go       # ServiceContainer
    ├── routing.go       # RoutingContainer
    ├── data.go          # DataContainer
    ├── telemetry.go     # TelemetryContainer
    └── ratelimit.go     # RateLimitContainer
```

## Consequences

### Positive
- Clear separation of concerns
- Easier to understand component boundaries
- Simpler dependency injection in tests
- Can initialize subsystems independently
- Better documentation through code structure

### Negative
- More files to navigate
- Requires updating imports when moving dependencies
- Slight increase in indirection

## Implementation
The refactoring was done in Phase 1.1 of the refactoring effort. All HTTP handlers and services were updated to use the appropriate sub-container.
