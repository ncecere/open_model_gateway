package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// ErrNotFound is returned when a resource is not found in the mock repository.
var ErrNotFound = errors.New("not found")

// MockTenantRepository is a mock implementation of TenantRepository for testing.
type MockTenantRepository struct {
	mu      sync.RWMutex
	tenants map[string]db.Tenant // keyed by ID string

	// Function hooks for custom behavior
	GetTenantByIDFunc         func(ctx context.Context, id pgtype.UUID) (db.Tenant, error)
	GetTenantByNameFunc       func(ctx context.Context, name string) (db.Tenant, error)
	ListTenantsFunc           func(ctx context.Context, params db.ListTenantsParams) ([]db.Tenant, error)
	ListPersonalTenantsFunc   func(ctx context.Context, params db.ListPersonalTenantsParams) ([]db.ListPersonalTenantsRow, error)
	CreateTenantFunc          func(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error)
	UpdateTenantNameFunc      func(ctx context.Context, params db.UpdateTenantNameParams) (db.Tenant, error)
	UpdateTenantStatusFunc    func(ctx context.Context, params db.UpdateTenantStatusParams) (db.Tenant, error)
}

// NewMockTenantRepository creates a new mock tenant repository.
func NewMockTenantRepository() *MockTenantRepository {
	return &MockTenantRepository{
		tenants: make(map[string]db.Tenant),
	}
}

func (m *MockTenantRepository) GetTenantByID(ctx context.Context, id pgtype.UUID) (db.Tenant, error) {
	if m.GetTenantByIDFunc != nil {
		return m.GetTenantByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.tenants[uuidToString(id)]; ok {
		return t, nil
	}
	return db.Tenant{}, ErrNotFound
}

func (m *MockTenantRepository) GetTenantByName(ctx context.Context, name string) (db.Tenant, error) {
	if m.GetTenantByNameFunc != nil {
		return m.GetTenantByNameFunc(ctx, name)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.tenants {
		if t.Name == name {
			return t, nil
		}
	}
	return db.Tenant{}, ErrNotFound
}

func (m *MockTenantRepository) ListTenants(ctx context.Context, params db.ListTenantsParams) ([]db.Tenant, error) {
	if m.ListTenantsFunc != nil {
		return m.ListTenantsFunc(ctx, params)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]db.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result, nil
}

func (m *MockTenantRepository) CreateTenant(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error) {
	if m.CreateTenantFunc != nil {
		return m.CreateTenantFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Generate a UUID for the new tenant
	newID := uuid.New()
	tenant := db.Tenant{
		ID:     pgtype.UUID{Bytes: newID, Valid: true},
		Name:   params.Name,
		Status: params.Status,
		Kind:   params.Kind,
	}
	m.tenants[newID.String()] = tenant
	return tenant, nil
}

func (m *MockTenantRepository) UpdateTenantStatus(ctx context.Context, params db.UpdateTenantStatusParams) (db.Tenant, error) {
	if m.UpdateTenantStatusFunc != nil {
		return m.UpdateTenantStatusFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := uuidToString(params.ID)
	if t, ok := m.tenants[key]; ok {
		t.Status = params.Status
		m.tenants[key] = t
		return t, nil
	}
	return db.Tenant{}, ErrNotFound
}

func (m *MockTenantRepository) ListPersonalTenants(ctx context.Context, params db.ListPersonalTenantsParams) ([]db.ListPersonalTenantsRow, error) {
	if m.ListPersonalTenantsFunc != nil {
		return m.ListPersonalTenantsFunc(ctx, params)
	}
	// Return empty list for mock
	return []db.ListPersonalTenantsRow{}, nil
}

func (m *MockTenantRepository) UpdateTenantName(ctx context.Context, params db.UpdateTenantNameParams) (db.Tenant, error) {
	if m.UpdateTenantNameFunc != nil {
		return m.UpdateTenantNameFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := uuidToString(params.ID)
	if t, ok := m.tenants[key]; ok {
		t.Name = params.Name
		m.tenants[key] = t
		return t, nil
	}
	return db.Tenant{}, ErrNotFound
}

// AddTenant adds a tenant to the mock store for testing.
func (m *MockTenantRepository) AddTenant(tenant db.Tenant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[uuidToString(tenant.ID)] = tenant
}

// MockTenantModelRepository is a mock implementation of TenantModelRepository for testing.
type MockTenantModelRepository struct {
	mu     sync.RWMutex
	models map[string][]string // keyed by tenant ID string

	ListTenantModelsFunc   func(ctx context.Context, tenantID pgtype.UUID) ([]string, error)
	InsertTenantModelFunc  func(ctx context.Context, params db.InsertTenantModelParams) error
	DeleteTenantModelsFunc func(ctx context.Context, tenantID pgtype.UUID) error
}

// NewMockTenantModelRepository creates a new mock tenant model repository.
func NewMockTenantModelRepository() *MockTenantModelRepository {
	return &MockTenantModelRepository{
		models: make(map[string][]string),
	}
}

func (m *MockTenantModelRepository) ListTenantModels(ctx context.Context, tenantID pgtype.UUID) ([]string, error) {
	if m.ListTenantModelsFunc != nil {
		return m.ListTenantModelsFunc(ctx, tenantID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if models, ok := m.models[uuidToString(tenantID)]; ok {
		return models, nil
	}
	return []string{}, nil
}

func (m *MockTenantModelRepository) InsertTenantModel(ctx context.Context, params db.InsertTenantModelParams) error {
	if m.InsertTenantModelFunc != nil {
		return m.InsertTenantModelFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := uuidToString(params.TenantID)
	m.models[key] = append(m.models[key], params.Alias)
	return nil
}

func (m *MockTenantModelRepository) DeleteTenantModels(ctx context.Context, tenantID pgtype.UUID) error {
	if m.DeleteTenantModelsFunc != nil {
		return m.DeleteTenantModelsFunc(ctx, tenantID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.models, uuidToString(tenantID))
	return nil
}

// SetTenantModels sets the models for a tenant in the mock store.
func (m *MockTenantModelRepository) SetTenantModels(tenantID pgtype.UUID, models []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models[uuidToString(tenantID)] = models
}

// MockApiKeyRepository is a mock implementation of ApiKeyRepository for testing.
type MockApiKeyRepository struct {
	mu       sync.RWMutex
	keys     map[string]db.ApiKey // keyed by ID string
	prefixes map[string]db.ApiKey // keyed by prefix

	GetAPIKeyByPrefixFunc   func(ctx context.Context, prefix string) (db.ApiKey, error)
	GetAPIKeyByIDFunc       func(ctx context.Context, id pgtype.UUID) (db.ApiKey, error)
	ListAPIKeysByTenantFunc func(ctx context.Context, tenantID pgtype.UUID) ([]db.ApiKey, error)
	CreateAPIKeyFunc        func(ctx context.Context, params db.CreateAPIKeyParams) (db.ApiKey, error)
	RevokeAPIKeyFunc        func(ctx context.Context, id pgtype.UUID) (db.ApiKey, error)
}

// NewMockApiKeyRepository creates a new mock API key repository.
func NewMockApiKeyRepository() *MockApiKeyRepository {
	return &MockApiKeyRepository{
		keys:     make(map[string]db.ApiKey),
		prefixes: make(map[string]db.ApiKey),
	}
}

func (m *MockApiKeyRepository) GetAPIKeyByPrefix(ctx context.Context, prefix string) (db.ApiKey, error) {
	if m.GetAPIKeyByPrefixFunc != nil {
		return m.GetAPIKeyByPrefixFunc(ctx, prefix)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if k, ok := m.prefixes[prefix]; ok {
		return k, nil
	}
	return db.ApiKey{}, ErrNotFound
}

func (m *MockApiKeyRepository) GetAPIKeyByID(ctx context.Context, id pgtype.UUID) (db.ApiKey, error) {
	if m.GetAPIKeyByIDFunc != nil {
		return m.GetAPIKeyByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if k, ok := m.keys[uuidToString(id)]; ok {
		return k, nil
	}
	return db.ApiKey{}, ErrNotFound
}

func (m *MockApiKeyRepository) ListAPIKeysByTenant(ctx context.Context, tenantID pgtype.UUID) ([]db.ApiKey, error) {
	if m.ListAPIKeysByTenantFunc != nil {
		return m.ListAPIKeysByTenantFunc(ctx, tenantID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []db.ApiKey
	tidStr := uuidToString(tenantID)
	for _, k := range m.keys {
		if uuidToString(k.TenantID) == tidStr {
			result = append(result, k)
		}
	}
	return result, nil
}

func (m *MockApiKeyRepository) CreateAPIKey(ctx context.Context, params db.CreateAPIKeyParams) (db.ApiKey, error) {
	if m.CreateAPIKeyFunc != nil {
		return m.CreateAPIKeyFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Generate a UUID for the new key
	newID := uuid.New()
	key := db.ApiKey{
		ID:          pgtype.UUID{Bytes: newID, Valid: true},
		TenantID:    params.TenantID,
		Prefix:      params.Prefix,
		SecretHash:  params.SecretHash,
		Name:        params.Name,
		ScopesJson:  params.ScopesJson,
		QuotaJson:   params.QuotaJson,
		Kind:        params.Kind,
		OwnerUserID: params.OwnerUserID,
	}
	m.keys[newID.String()] = key
	m.prefixes[params.Prefix] = key
	return key, nil
}

func (m *MockApiKeyRepository) RevokeAPIKey(ctx context.Context, id pgtype.UUID) (db.ApiKey, error) {
	if m.RevokeAPIKeyFunc != nil {
		return m.RevokeAPIKeyFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := uuidToString(id)
	if k, ok := m.keys[key]; ok {
		k.RevokedAt.Valid = true
		m.keys[key] = k
		return k, nil
	}
	return db.ApiKey{}, ErrNotFound
}

// AddAPIKey adds an API key to the mock store for testing.
func (m *MockApiKeyRepository) AddAPIKey(key db.ApiKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[uuidToString(key.ID)] = key
	m.prefixes[key.Prefix] = key
}

// MockUsageRepository is a mock implementation of UsageRepository for testing.
type MockUsageRepository struct {
	mu      sync.RWMutex
	records []db.UsageRecord

	InsertUsageRecordFunc func(ctx context.Context, params db.InsertUsageRecordParams) (db.UsageRecord, error)
	SumUsageForTenantFunc func(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error)
}

// NewMockUsageRepository creates a new mock usage repository.
func NewMockUsageRepository() *MockUsageRepository {
	return &MockUsageRepository{
		records: make([]db.UsageRecord, 0),
	}
}

func (m *MockUsageRepository) InsertUsageRecord(ctx context.Context, params db.InsertUsageRecordParams) (db.UsageRecord, error) {
	if m.InsertUsageRecordFunc != nil {
		return m.InsertUsageRecordFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	record := db.UsageRecord{
		TenantID:      params.TenantID,
		ApiKeyID:      params.ApiKeyID,
		Ts:            params.Ts,
		ModelAlias:    params.ModelAlias,
		Provider:      params.Provider,
		InputTokens:   params.InputTokens,
		OutputTokens:  params.OutputTokens,
		Requests:      params.Requests,
		CostCents:     params.CostCents,
		CostUsdMicros: params.CostUsdMicros,
	}
	m.records = append(m.records, record)
	return record, nil
}

func (m *MockUsageRepository) SumUsageForTenant(ctx context.Context, params db.SumUsageForTenantParams) (db.SumUsageForTenantRow, error) {
	if m.SumUsageForTenantFunc != nil {
		return m.SumUsageForTenantFunc(ctx, params)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return empty summary for mock
	return db.SumUsageForTenantRow{}, nil
}

// MockBudgetRepository is a mock implementation of BudgetRepository for testing.
type MockBudgetRepository struct {
	mu      sync.RWMutex
	budgets map[string]db.TenantBudgetOverride // keyed by tenant ID string

	GetTenantBudgetOverrideFunc    func(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error)
	ListTenantBudgetOverridesFunc  func(ctx context.Context) ([]db.TenantBudgetOverride, error)
	UpsertTenantBudgetOverrideFunc func(ctx context.Context, params db.UpsertTenantBudgetOverrideParams) (db.TenantBudgetOverride, error)
	DeleteTenantBudgetOverrideFunc func(ctx context.Context, tenantID pgtype.UUID) error
}

// NewMockBudgetRepository creates a new mock budget repository.
func NewMockBudgetRepository() *MockBudgetRepository {
	return &MockBudgetRepository{
		budgets: make(map[string]db.TenantBudgetOverride),
	}
}

func (m *MockBudgetRepository) GetTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) (db.TenantBudgetOverride, error) {
	if m.GetTenantBudgetOverrideFunc != nil {
		return m.GetTenantBudgetOverrideFunc(ctx, tenantID)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if b, ok := m.budgets[uuidToString(tenantID)]; ok {
		return b, nil
	}
	return db.TenantBudgetOverride{}, ErrNotFound
}

func (m *MockBudgetRepository) ListTenantBudgetOverrides(ctx context.Context) ([]db.TenantBudgetOverride, error) {
	if m.ListTenantBudgetOverridesFunc != nil {
		return m.ListTenantBudgetOverridesFunc(ctx)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]db.TenantBudgetOverride, 0, len(m.budgets))
	for _, b := range m.budgets {
		result = append(result, b)
	}
	return result, nil
}

func (m *MockBudgetRepository) UpsertTenantBudgetOverride(ctx context.Context, params db.UpsertTenantBudgetOverrideParams) (db.TenantBudgetOverride, error) {
	if m.UpsertTenantBudgetOverrideFunc != nil {
		return m.UpsertTenantBudgetOverrideFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	budget := db.TenantBudgetOverride{
		TenantID:         params.TenantID,
		BudgetUsd:        params.BudgetUsd,
		WarningThreshold: params.WarningThreshold,
	}
	m.budgets[uuidToString(params.TenantID)] = budget
	return budget, nil
}

func (m *MockBudgetRepository) DeleteTenantBudgetOverride(ctx context.Context, tenantID pgtype.UUID) error {
	if m.DeleteTenantBudgetOverrideFunc != nil {
		return m.DeleteTenantBudgetOverrideFunc(ctx, tenantID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.budgets, uuidToString(tenantID))
	return nil
}

// MockAuditRepository is a mock implementation of AuditRepository for testing.
type MockAuditRepository struct {
	mu     sync.RWMutex
	events []db.AdminAuditLog

	InsertAuditLogFunc func(ctx context.Context, params db.InsertAuditLogParams) (db.AdminAuditLog, error)
	ListAuditLogsFunc  func(ctx context.Context, params db.ListAuditLogsParams) ([]db.AdminAuditLog, error)
}

// NewMockAuditRepository creates a new mock audit repository.
func NewMockAuditRepository() *MockAuditRepository {
	return &MockAuditRepository{
		events: make([]db.AdminAuditLog, 0),
	}
}

func (m *MockAuditRepository) InsertAuditLog(ctx context.Context, params db.InsertAuditLogParams) (db.AdminAuditLog, error) {
	if m.InsertAuditLogFunc != nil {
		return m.InsertAuditLogFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	event := db.AdminAuditLog{
		UserID:       params.UserID,
		Action:       params.Action,
		ResourceType: params.ResourceType,
		ResourceID:   params.ResourceID,
		Metadata:     params.Metadata,
	}
	m.events = append(m.events, event)
	return event, nil
}

func (m *MockAuditRepository) ListAuditLogs(ctx context.Context, params db.ListAuditLogsParams) ([]db.AdminAuditLog, error) {
	if m.ListAuditLogsFunc != nil {
		return m.ListAuditLogsFunc(ctx, params)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.events, nil
}

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	mu    sync.RWMutex
	users map[string]db.User // keyed by ID string

	GetUserByIDFunc    func(ctx context.Context, id pgtype.UUID) (db.User, error)
	GetUserByEmailFunc func(ctx context.Context, email string) (db.User, error)
	ListUsersFunc      func(ctx context.Context, params db.ListUsersParams) ([]db.User, error)
	CreateUserFunc     func(ctx context.Context, params db.CreateUserParams) (db.User, error)
}

// NewMockUserRepository creates a new mock user repository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]db.User),
	}
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, id)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if u, ok := m.users[uuidToString(id)]; ok {
		return u, nil
	}
	return db.User{}, ErrNotFound
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if m.GetUserByEmailFunc != nil {
		return m.GetUserByEmailFunc(ctx, email)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return db.User{}, ErrNotFound
}

func (m *MockUserRepository) ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error) {
	if m.ListUsersFunc != nil {
		return m.ListUsersFunc(ctx, params)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]db.User, 0, len(m.users))
	for _, u := range m.users {
		result = append(result, u)
	}
	return result, nil
}

func (m *MockUserRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Generate new UUID for user ID (database generates this)
	newID := uuid.New()
	userID := pgtype.UUID{Bytes: newID, Valid: true}
	user := db.User{
		ID:    userID,
		Email: params.Email,
		Name:  params.Name,
	}
	m.users[uuidToString(userID)] = user
	return user, nil
}

// AddUser adds a user to the mock store for testing.
func (m *MockUserRepository) AddUser(user db.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[uuidToString(user.ID)] = user
}

// Helper function to convert pgtype.UUID to string
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	u, err := uuid.FromBytes(id.Bytes[:])
	if err != nil {
		return ""
	}
	return u.String()
}

// Compile-time checks that mocks implement interfaces
var (
	_ TenantRepository      = (*MockTenantRepository)(nil)
	_ TenantModelRepository = (*MockTenantModelRepository)(nil)
	_ ApiKeyRepository      = (*MockApiKeyRepository)(nil)
	_ UsageRepository       = (*MockUsageRepository)(nil)
	_ BudgetRepository      = (*MockBudgetRepository)(nil)
	_ AuditRepository       = (*MockAuditRepository)(nil)
	_ UserRepository        = (*MockUserRepository)(nil)
)
