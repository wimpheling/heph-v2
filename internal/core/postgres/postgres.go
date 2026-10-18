// Package postgres implements opinionated core PostgreSQL persistence.
package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	a "github.com/wimpheling/heph-v2/internal/core/authorization"
	"github.com/wimpheling/heph-v2/internal/core/identity"
)

//go:embed migrations/*.sql
var migrations embed.FS
var schemaName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Store struct {
	pool          *pgxpool.Pool
	db            queryer
	inTransaction bool
}

func Open(ctx context.Context, url, schema string) (*Store, error) {
	if !schemaName.MatchString(schema) {
		return nil, fmt.Errorf("invalid database schema name")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid database connection settings")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("database pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database readiness: %w", err)
	}
	return &Store{pool: pool, db: pool}, nil
}
func (s *Store) Close() { s.pool.Close() }
func (s *Store) Migrate(ctx context.Context, schema string) error {
	return s.Transaction(ctx, func(store a.Store) error {
		tx := store.(*Store)
		if _, err := tx.db.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+pgx.Identifier{schema}.Sanitize()); err != nil {
			return err
		}
		if _, err := tx.db.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY)"); err != nil {
			return err
		}
		var applied bool
		if err := tx.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=1)").Scan(&applied); err != nil {
			return err
		}
		if applied {
			return nil
		}
		sql, err := migrations.ReadFile("migrations/001_authorization.sql")
		if err != nil {
			return err
		}
		if _, err = tx.db.Exec(ctx, string(sql)); err != nil {
			return err
		}
		_, err = tx.db.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES(1)")
		return err
	})
}
func (s *Store) Reset(ctx context.Context) error {
	return s.Transaction(ctx, func(store a.Store) error {
		_, err := store.(*Store).db.Exec(ctx, "DROP TABLE IF EXISTS bootstrap_state, grants, scopes, schema_migrations CASCADE")
		return err
	})
}
func storageError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return a.ErrNotFound
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return a.ErrConflict
	}
	return fmt.Errorf("%w: %v", a.ErrUnavailable, err)
}
func (s *Store) Transaction(ctx context.Context, fn func(a.Store) error) error {
	if s.inTransaction {
		return fn(s)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return storageError(err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx) // Rollback after commit is harmless.
	}()
	// Every authorization decision and mutation uses this lock before reading.
	// Read-committed snapshots then observe every preceding committed revocation.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(728492317)"); err != nil {
		return storageError(err)
	}
	if err = fn(&Store{pool: s.pool, db: tx, inTransaction: true}); err != nil {
		return err
	}
	return storageError(tx.Commit(ctx))
}

func (s *Store) Scope(ctx context.Context, ref a.ScopeRef) (a.Scope, error) {
	scope := a.Scope{Ref: ref}
	var pk, pid, source, subject *string
	err := s.db.QueryRow(ctx, "SELECT parent_kind,parent_id,owner_source,owner_subject,resource_kind,ref_name FROM scopes WHERE kind=$1 AND id=$2", ref.Kind, ref.ID).Scan(&pk, &pid, &source, &subject, &scope.ResourceKind, &scope.RefName)
	if err != nil {
		return scope, storageError(err)
	}
	if pk != nil && pid != nil {
		scope.Parent = &a.ScopeRef{Kind: a.ScopeKind(*pk), ID: a.ScopeID(*pid)}
	}
	if source != nil && subject != nil {
		scope.Owner = &identity.ActorRef{Source: *source, Subject: *subject}
	}
	return scope, nil
}

const grantColumns = "id,grantee_source,grantee_subject,action,scope_kind,scope_id,granted_by_source,granted_by_subject,created_at,revoked_by_source,revoked_by_subject,revoked_at"

type scanner interface{ Scan(...any) error }

func scanGrant(row scanner) (a.GrantRecord, error) {
	var r a.GrantRecord
	var source, subject *string
	rev := a.Revocation{}
	var at *time.Time
	err := row.Scan(&r.Grant.ID, &r.Grant.Grantee.Source, &r.Grant.Grantee.Subject, &r.Grant.Action, &r.Grant.Scope.Kind, &r.Grant.Scope.ID, &r.Grant.GrantedBy.Source, &r.Grant.GrantedBy.Subject, &r.Grant.CreatedAt, &source, &subject, &at)
	if err != nil {
		return r, storageError(err)
	}
	if at != nil && source != nil && subject != nil {
		rev.GrantID = r.Grant.ID
		rev.RevokedBy = identity.ActorRef{Source: *source, Subject: *subject}
		rev.RevokedAt = *at
		r.Revocation = &rev
	}
	return r, nil
}
func (s *Store) Get(ctx context.Context, id a.GrantID) (a.GrantRecord, error) {
	return scanGrant(s.db.QueryRow(ctx, "SELECT "+grantColumns+" FROM grants WHERE id=$1", id))
}
func (s *Store) ActiveGrants(ctx context.Context, actor identity.ActorRef) ([]a.Grant, error) {
	rows, err := s.db.Query(ctx, "SELECT "+grantColumns+" FROM grants WHERE grantee_source=$1 AND grantee_subject=$2 AND revoked_at IS NULL", actor.Source, actor.Subject)
	if err != nil {
		return nil, storageError(err)
	}
	defer rows.Close()
	grants := []a.Grant{}
	for rows.Next() {
		r, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		grants = append(grants, r.Grant)
	}
	return grants, storageError(rows.Err())
}
func (s *Store) List(ctx context.Context, ref a.ScopeRef, limit int, after string) (a.GrantPage, error) {
	page := a.GrantPage{Grants: []a.Grant{}}
	rows, err := s.db.Query(ctx, "SELECT "+grantColumns+" FROM grants WHERE scope_kind=$1 AND scope_id=$2 AND id>$3 AND revoked_at IS NULL ORDER BY id LIMIT $4", ref.Kind, ref.ID, after, limit+1)
	if err != nil {
		return page, storageError(err)
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanGrant(rows)
		if err != nil {
			return page, err
		}
		page.Grants = append(page.Grants, r.Grant)
	}
	if err = rows.Err(); err != nil {
		return page, storageError(err)
	}
	if len(page.Grants) > limit {
		page.Grants = page.Grants[:limit]
		page.NextPageToken = string(page.Grants[limit-1].ID)
	}
	return page, nil
}
func (s *Store) Create(ctx context.Context, actor identity.ActorRef, input a.CreateInput) (a.Grant, error) {
	id := a.GrantID(rand.Text())
	r, err := scanGrant(s.db.QueryRow(ctx, "INSERT INTO grants(id,grantee_source,grantee_subject,action,scope_kind,scope_id,granted_by_source,granted_by_subject) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "+grantColumns, id, input.Grantee.Source, input.Grantee.Subject, input.Action, input.Scope.Kind, input.Scope.ID, actor.Source, actor.Subject))
	return r.Grant, err
}
func (s *Store) Revoke(ctx context.Context, actor identity.ActorRef, id a.GrantID) (a.Revocation, error) {
	r, err := scanGrant(s.db.QueryRow(ctx, "UPDATE grants SET revoked_by_source=COALESCE(revoked_by_source,$2),revoked_by_subject=COALESCE(revoked_by_subject,$3),revoked_at=COALESCE(revoked_at,clock_timestamp()) WHERE id=$1 RETURNING "+grantColumns, id, actor.Source, actor.Subject))
	if err != nil {
		return a.Revocation{}, err
	}
	return *r.Revocation, nil
}

func (s *Store) Bootstrap(ctx context.Context, actors []identity.ActorRef) error {
	if len(actors) == 0 {
		return a.ErrInvalid
	}
	return s.Transaction(ctx, func(store a.Store) error {
		tx := store.(*Store)
		if _, err := tx.db.Exec(ctx, "INSERT INTO scopes(kind,id) VALUES('platform','platform') ON CONFLICT DO NOTHING"); err != nil {
			return storageError(err)
		}
		for _, actor := range actors {
			sum := sha256.Sum256([]byte(actor.Source + "\x00" + actor.Subject))
			id := fmt.Sprintf("personal_%x", sum[:16])
			if _, err := tx.db.Exec(ctx, "INSERT INTO scopes(kind,id,parent_kind,parent_id,owner_source,owner_subject) VALUES('personal',$1,'platform','platform',$2,$3) ON CONFLICT DO NOTHING", id, actor.Source, actor.Subject); err != nil {
				return storageError(err)
			}
		}
		var source, subject string
		err := tx.db.QueryRow(ctx, "SELECT actor_source,actor_subject FROM bootstrap_state WHERE singleton=true").Scan(&source, &subject)
		if err == nil {
			if source != actors[0].Source || subject != actors[0].Subject {
				return fmt.Errorf("bootstrap actor changed; reset the POC database to initialize another actor")
			}
			return nil // Never restore permissions that were revoked through RPC.
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return storageError(err)
		}
		for _, action := range a.Actions() {
			if _, err := tx.Create(ctx, actors[0], a.CreateInput{Grantee: actors[0], Action: action, Scope: a.PlatformScope}); err != nil {
				return err
			}
		}
		_, err = tx.db.Exec(ctx, "INSERT INTO bootstrap_state(singleton,actor_source,actor_subject) VALUES(true,$1,$2)", actors[0].Source, actors[0].Subject)
		return storageError(err)
	})
}
