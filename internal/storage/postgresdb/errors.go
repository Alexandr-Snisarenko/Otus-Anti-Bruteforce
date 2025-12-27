package postgresdb

import "errors"

var (
	// ErrNoRows обозначает отсутствие записей в результате запроса.
	// ErrNoRows = sqlx.ErrNotFound
	// ErrDuplicateKey обозначает нарушение уникального ограничения.
	ErrDuplicateKey = errors.New("duplicate key value violates unique constraint")
	// ErrEmptyCIDR обозначает пустое значение CIDR.
	ErrEmptyCIDR = errors.New("cidr is empty")
)
