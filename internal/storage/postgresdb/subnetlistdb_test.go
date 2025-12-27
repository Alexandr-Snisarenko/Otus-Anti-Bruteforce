package postgresdb

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func setup(t *testing.T) (*SubnetListDB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sx := sqlx.NewDb(db, "sqlmock")
	s := &SubnetListDB{db: sx}
	cleanup := func() {
		sx.Close()
	}
	return s, mock, cleanup
}

func TestGetSubnetLists_Success(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"cidr"}).AddRow("1.2.3.0/24").AddRow("10.0.0.0/8")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT cidr\n\tFROM subnets\n\tWHERE list_type = $1")).
		WithArgs(domain.Whitelist).WillReturnRows(rows)

	got, err := s.GetSubnetLists(context.Background(), domain.Whitelist)
	if err != nil {
		t.Fatalf("GetSubnetLists error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetSubnetLists_Error(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	mock.ExpectQuery("SELECT cidr").WithArgs(domain.Whitelist).WillReturnError(errors.New("db error"))

	_, err := s.GetSubnetLists(context.Background(), domain.Whitelist)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestSaveSubnetList_Empty_NoExec(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	// no expectation: SaveSubnetList with empty slice should be no-op
	if err := s.SaveSubnetList(context.Background(), domain.Whitelist, []string{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSaveSubnetList_Exec(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	mock.ExpectExec("INSERT INTO subnets").WillReturnResult(sqlmock.NewResult(1, 2))

	if err := s.SaveSubnetList(context.Background(), domain.Whitelist, []string{"1.2.3.0/24", "10.0.0.0/8"}); err != nil {
		t.Fatalf("SaveSubnetList error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestClearSubnetList_Exec(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM subnets\n    WHERE list_type = $1")).
		WithArgs(domain.Whitelist).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := s.ClearSubnetList(context.Background(), domain.Whitelist); err != nil {
		t.Fatalf("ClearSubnetList error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestAddCIDRToSubnetList_Empty(t *testing.T) {
	s, _, cleanup := setup(t)
	defer cleanup()

	if err := s.AddCIDRToSubnetList(context.Background(), domain.Whitelist, ""); err == nil {
		t.Fatalf("expected ErrEmptyCIDR, got nil")
	}
}

func TestAddCIDRToSubnetList_Exec(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	mock.ExpectExec("INSERT INTO subnets").WillReturnResult(sqlmock.NewResult(1, 1))

	if err := s.AddCIDRToSubnetList(context.Background(), domain.Whitelist, "1.2.3.0/24"); err != nil {
		t.Fatalf("AddCIDRToSubnetList error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRemoveCIDRFromSubnetList_Empty(t *testing.T) {
	s, _, cleanup := setup(t)
	defer cleanup()

	if err := s.RemoveCIDRFromSubnetList(context.Background(), domain.Whitelist, ""); err == nil {
		t.Fatalf("expected ErrEmptyCIDR, got nil")
	}
}

func TestRemoveCIDRFromSubnetList_Exec(t *testing.T) {
	s, mock, cleanup := setup(t)
	defer cleanup()

	mock.ExpectExec("DELETE FROM subnets").WillReturnResult(sqlmock.NewResult(1, 1))

	if err := s.RemoveCIDRFromSubnetList(context.Background(), domain.Whitelist, "1.2.3.0/24"); err != nil {
		t.Fatalf("RemoveCIDRFromSubnetList error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
