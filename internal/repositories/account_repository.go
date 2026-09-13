package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/Neue-Konzepte-BaaS/backend/internal/models"
	database "github.com/Neue-Konzepte-BaaS/backend/internal/repositories/db"
	"github.com/Neue-Konzepte-BaaS/backend/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolation is the Postgres SQLSTATE code for a unique-constraint breach;
// on the account table that means the email is already registered.
const uniqueViolation = "23505"

type accountRepository struct {
	pool    *pgxpool.Pool
	queries *database.Queries
}

func NewAccountRepository(pool *pgxpool.Pool, queries *database.Queries) services.AccountRepository {
	return &accountRepository{pool: pool, queries: queries}
}

func (r *accountRepository) GetAccountByEmail(ctx context.Context, email string) (models.Account, error) {
	row, err := r.queries.GetAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Account{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		} else {
			return models.Account{}, err
		}
	}

	return models.Account{
		ID:           row.ID,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         models.Role(row.Role),
	}, nil
}

func (r *accountRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (models.Account, error) {
	row, err := r.queries.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Account{}, fmt.Errorf("db error: %w %w", err, services.ErrNotFound)
		} else {
			return models.Account{}, err
		}
	}

	return models.Account{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		Role:      models.Role(row.Role),
	}, nil
}

func (r *accountRepository) CreateFarmer(ctx context.Context, account models.Account, farmName string, postalCode int32) (models.Account, error) {
	return r.createAccountWithSubtype(ctx, account, models.RoleFarmer, func(ctx context.Context, q *database.Queries, id uuid.UUID) error {
		return q.InsertFarmer(ctx, database.InsertFarmerParams{
			AccountID:  id,
			FarmName:   farmName,
			PostalCode: postalCode,
		})
	})
}

func (r *accountRepository) CreateCustomer(ctx context.Context, account models.Account, postalCode int32) (models.Account, error) {
	return r.createAccountWithSubtype(ctx, account, models.RoleCustomer, func(ctx context.Context, q *database.Queries, id uuid.UUID) error {
		return q.InsertCustomer(ctx, database.InsertCustomerParams{
			AccountID:  id,
			PostalCode: postalCode,
		})
	})
}

// createAccountWithSubtype inserts the account row and its role-specific subtype
// row in a single transaction, so an account can never exist without the
// membership that gives it a role. insertSubtype receives the tx-bound queries
// and the new account id.
func (r *accountRepository) createAccountWithSubtype(
	ctx context.Context,
	account models.Account,
	role models.Role,
	insertSubtype func(ctx context.Context, q *database.Queries, id uuid.UUID) error,
) (models.Account, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Account{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	qtx := r.queries.WithTx(tx)

	id, err := qtx.InsertAccount(ctx, database.InsertAccountParams{
		FirstName:    account.FirstName,
		LastName:     account.LastName,
		Email:        account.Email,
		PasswordHash: account.PasswordHash,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return models.Account{}, services.ErrEmailTaken
		}
		return models.Account{}, fmt.Errorf("inserting account: %w", err)
	}

	if err := insertSubtype(ctx, qtx, id); err != nil {
		return models.Account{}, fmt.Errorf("inserting %s: %w", role, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Account{}, fmt.Errorf("commit tx: %w", err)
	}

	account.ID = id
	account.Role = role
	account.PasswordHash = ""
	return account, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}
