# ADR 002: Service Interface Pattern

## Status
Accepted

## Context
Services directly depended on `*db.Queries` (SQLC-generated types), making them tightly coupled to the database implementation and difficult to unit test without a real database.

## Decision
Introduce repository interfaces between services and the database layer:

1. Define repository interfaces in `internal/repository/interfaces.go`
2. Create `QueriesAdapter` that wraps `*db.Queries` and implements all repository interfaces
3. Update services to depend on repository interfaces instead of concrete types
4. Generate mock implementations for testing

## Repository Interfaces
```go
// TenantRepository handles tenant data operations
type TenantRepository interface {
    CreateTenant(ctx, params) (Tenant, error)
    GetTenantByID(ctx, id) (Tenant, error)
    ListTenants(ctx, params) ([]Tenant, error)
    // ...
}

// UsageRepository handles usage record operations
type UsageRepository interface {
    InsertUsageRecord(ctx, params) (UsageRecord, error)
    AggregateUsage(ctx, params) ([]UsageRow, error)
    // ...
}
```

## Structure
```
internal/
├── repository/
│   ├── interfaces.go    # All repository interfaces
│   └── adapter.go       # QueriesAdapter implementation
└── services/
    └── tenant/
        └── service.go   # Depends on TenantRepository interface
```

## Consequences

### Positive
- Services can be unit tested with mock repositories
- Clear contracts between services and data layer
- Easier to swap database implementations
- Better separation of concerns
- Enables parallel test execution

### Negative
- Additional abstraction layer
- Need to maintain interface and adapter in sync with SQLC
- Slightly more code to write

## Testing
With repository interfaces, services can be tested using mock implementations:

```go
type mockTenantRepo struct {
    tenants map[uuid.UUID]Tenant
}

func (m *mockTenantRepo) GetTenantByID(ctx, id) (Tenant, error) {
    if t, ok := m.tenants[id]; ok {
        return t, nil
    }
    return Tenant{}, ErrNotFound
}
```

## Implementation
The refactoring was done in Phase 1.4 of the refactoring effort. All services were updated to depend on repository interfaces.
