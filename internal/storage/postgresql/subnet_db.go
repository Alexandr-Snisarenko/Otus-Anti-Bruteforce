package postgresql

import (
	"context"
	"errors"
	"fmt"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	_ "github.com/jackc/pgx/v4/stdlib" // register pgx driver
	"github.com/jmoiron/sqlx"
)

type PgStorage struct {
	db *sqlx.DB
}

func (s *PgStorage) GetSubnetLists(ctx context.Context, listType domain.ListType) ([]string, error) {
	const query = `
	SELECT cidr
	FROM subnets
	WHERE list_type = $1`
	var cidrs []string
	err := s.db.SelectContext(ctx, &cidrs, query, listType)
	if err != nil {
		return nil, err
	}
	return cidrs, nil
}

func (s *PgStorage) SaveSubnetList(ctx context.Context, listType domain.ListType, cidrs []string) error {
	if len(cidrs) == 0 {
		return nil
	}
	const insertQuery = `
    INSERT INTO subnets (cidr, list_type)
    VALUES (:cidr, :list_type)
    ON CONFLICT (cidr, list_type) DO NOTHING;
`
	rows := make([]subnetRow, 0, len(cidrs))
	for _, cidr := range cidrs {
		rows = append(rows, subnetRow{
			cidr:     cidr,
			listType: string(listType),
		})
	}

	_, err := s.db.NamedExecContext(ctx, insertQuery, rows)
	return err
}

func (s *PgStorage) ClearSubnetList(ctx context.Context, listType domain.ListType) error {
	const query = `
    DELETE FROM subnets
    WHERE list_type = $1`

	// На количество строк не проверяем, факт непосредственного удаления не важен
	_, err := s.db.ExecContext(ctx, query, listType)
	return err
}

func (s *PgStorage) AddCIDRToSubnetList(ctx context.Context, listType domain.ListType, cidr string) error {
	if cidr == "" {
		return ErrEmptyCIDR
	}
	const query = `
    INSERT INTO subnets (cidr, list_type)
    VALUES (:cidr, :list_type)
    ON CONFLICT (cidr, list_type) DO NOTHING`

	// На количество строк не проверяем, если есть дубликат - считаем, что операция успешна
	_, err := s.db.NamedExecContext(ctx, query, subnetRow{cidr: cidr, listType: string(listType)})
	return err
}

func (s *PgStorage) RemoveCIDRFromSubnetList(ctx context.Context, listType domain.ListType, cidr string) error {
	if cidr == "" {
		return ErrEmptyCIDR
	}
	const query = `
    DELETE FROM subnets
    WHERE cidr = :cidr 
	AND list_type = :list_type`

	// На количество строк не проверяем, факт непосредственного удаления не важен
	_, err := s.db.NamedExecContext(ctx, query, subnetRow{cidr: cidr, listType: string(listType)})
	return err
}

func (s *PgStorage) Close() error {
	return s.db.Close()
}

func New(cfg config.Database) (*PgStorage, error) {
	db, err := OpenDB(cfg)
	if err != nil {
		return nil, err
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(cfg.Postgresql.Pool.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Postgresql.Pool.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Postgresql.Pool.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.Postgresql.Pool.ConnMaxIdleTime)

	return &PgStorage{db: db}, nil
}

func OpenDB(cfg config.Database) (*sqlx.DB, error) {
	// Если заполнен параметр конфиге "dns", то используем его.
	// В этом случае параметры User и т.д. - игнорируются
	dsn := cfg.Postgresql.Dsn
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.Postgresql.User, cfg.Postgresql.Password,
			cfg.Postgresql.Host, cfg.Postgresql.Port, cfg.Postgresql.Name)
		if dsn == "" {
			return nil, errors.New("empty DSN")
		}
	}

	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
