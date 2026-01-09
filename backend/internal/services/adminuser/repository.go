package adminuser

import (
	"context"

	"github.com/ncecere/open_model_gateway/backend/internal/db"
)

// Repository defines the data access interface for admin user operations.
type Repository interface {
	ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)
}

// queriesAdapter wraps *db.Queries to implement the Repository interface.
type queriesAdapter struct {
	q *db.Queries
}

// NewQueriesRepository creates a Repository backed by *db.Queries.
func NewQueriesRepository(q *db.Queries) Repository {
	return &queriesAdapter{q: q}
}

func (a *queriesAdapter) ListUsers(ctx context.Context, params db.ListUsersParams) ([]db.User, error) {
	return a.q.ListUsers(ctx, params)
}

func (a *queriesAdapter) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return a.q.GetUserByEmail(ctx, email)
}

func (a *queriesAdapter) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	return a.q.CreateUser(ctx, params)
}
