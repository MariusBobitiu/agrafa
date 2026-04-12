package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type AuthRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewAuthRepository(db *sql.DB, queries *generated.Queries) *AuthRepository {
	return &AuthRepository{
		db:      db,
		queries: queries,
	}
}

func (r *AuthRepository) Register(
	ctx context.Context,
	userParams generated.CreateUserParams,
	passwordParams generated.CreatePasswordCredentialParams,
	sessionParams generated.CreateSessionParams,
) (generated.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return generated.User{}, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	queries := r.queries.WithTx(tx)

	user, err := queries.CreateUser(ctx, userParams)
	if err != nil {
		return generated.User{}, err
	}

	if _, err := queries.CreatePasswordCredential(ctx, passwordParams); err != nil {
		return generated.User{}, err
	}

	if _, err := queries.CreateSession(ctx, sessionParams); err != nil {
		return generated.User{}, err
	}

	if err := tx.Commit(); err != nil {
		return generated.User{}, fmt.Errorf("commit tx: %w", err)
	}

	tx = nil
	return user, nil
}

func (r *AuthRepository) GetUserWithPasswordByEmail(ctx context.Context, email string) (generated.GetUserWithPasswordByEmailRow, error) {
	return r.queries.GetUserWithPasswordByEmail(ctx, email)
}

func (r *AuthRepository) MarkEmailVerifiedByID(ctx context.Context, id string) error {
	rowsUpdated, err := r.queries.MarkUserEmailVerifiedByID(ctx, id)
	if err != nil {
		return err
	}
	if rowsUpdated == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *AuthRepository) GetUserWithPasswordByID(ctx context.Context, id string) (generated.GetUserWithPasswordByIDRow, error) {
	return r.queries.GetUserWithPasswordByID(ctx, id)
}

func (r *AuthRepository) CompleteOnboardingByUserID(ctx context.Context, id string) (generated.User, error) {
	return r.queries.CompleteUserOnboardingByID(ctx, id)
}

func (r *AuthRepository) CreateSession(ctx context.Context, params generated.CreateSessionParams) (generated.Session, error) {
	return r.queries.CreateSession(ctx, params)
}

func (r *AuthRepository) GetSessionUserByTokenHash(ctx context.Context, tokenHash string) (generated.GetSessionUserByTokenHashRow, error) {
	return r.queries.GetSessionUserByTokenHash(ctx, tokenHash)
}

func (r *AuthRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (generated.Session, error) {
	return r.queries.GetSessionByTokenHash(ctx, tokenHash)
}

func (r *AuthRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return r.queries.DeleteSessionByTokenHash(ctx, tokenHash)
}

func (r *AuthRepository) ListSessionsByUserID(ctx context.Context, userID string) ([]generated.Session, error) {
	return r.queries.ListSessionsByUserID(ctx, userID)
}

func (r *AuthRepository) DeleteSessionByIDAndUser(ctx context.Context, id string, userID string) (int64, error) {
	return r.queries.DeleteSessionByIDAndUser(ctx, generated.DeleteSessionByIDAndUserParams{
		ID:     id,
		UserID: userID,
	})
}

func (r *AuthRepository) DeleteOtherSessionsByUser(ctx context.Context, userID string, currentTokenHash string) (int64, error) {
	return r.queries.DeleteOtherSessionsByUser(ctx, generated.DeleteOtherSessionsByUserParams{
		UserID:    userID,
		TokenHash: currentTokenHash,
	})
}

func (r *AuthRepository) ReplaceVerificationToken(ctx context.Context, params generated.CreateVerificationTokenParams) (generated.VerificationToken, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return generated.VerificationToken{}, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	queries := r.queries.WithTx(tx)
	if _, err := queries.DeleteVerificationTokensByIdentifierAndType(ctx, generated.DeleteVerificationTokensByIdentifierAndTypeParams{
		Identifier: params.Identifier,
		Type:       params.Type,
	}); err != nil {
		return generated.VerificationToken{}, err
	}

	token, err := queries.CreateVerificationToken(ctx, params)
	if err != nil {
		return generated.VerificationToken{}, err
	}

	if err := tx.Commit(); err != nil {
		return generated.VerificationToken{}, fmt.Errorf("commit tx: %w", err)
	}

	tx = nil
	return token, nil
}

func (r *AuthRepository) GetVerificationTokenByTokenHashAndType(ctx context.Context, tokenHash string, tokenType string) (generated.VerificationToken, error) {
	return r.queries.GetVerificationTokenByTokenHashAndType(ctx, generated.GetVerificationTokenByTokenHashAndTypeParams{
		TokenHash: tokenHash,
		Type:      tokenType,
	})
}

func (r *AuthRepository) DeleteVerificationTokenByID(ctx context.Context, tokenID string) (int64, error) {
	return r.queries.DeleteVerificationTokenByID(ctx, tokenID)
}

func (r *AuthRepository) ConfirmEmailVerification(ctx context.Context, tokenID string, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	queries := r.queries.WithTx(tx)
	rowsDeleted, err := queries.DeleteVerificationTokenByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if rowsDeleted == 0 {
		return sql.ErrNoRows
	}

	rowsUpdated, err := queries.MarkUserEmailVerifiedByID(ctx, userID)
	if err != nil {
		return err
	}
	if rowsUpdated == 0 {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	tx = nil
	return nil
}

func (r *AuthRepository) ResetPasswordWithToken(ctx context.Context, tokenID string, userID string, passwordHash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	queries := r.queries.WithTx(tx)
	rowsDeleted, err := queries.DeleteVerificationTokenByID(ctx, tokenID)
	if err != nil {
		return err
	}
	if rowsDeleted == 0 {
		return sql.ErrNoRows
	}

	rowsUpdated, err := queries.UpdatePasswordCredentialByUserID(ctx, generated.UpdatePasswordCredentialByUserIDParams{
		UserID:       userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return err
	}
	if rowsUpdated == 0 {
		return sql.ErrNoRows
	}

	if _, err := queries.DeleteSessionsByUserID(ctx, userID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	tx = nil
	return nil
}
