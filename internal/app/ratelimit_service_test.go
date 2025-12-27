package app

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	mem "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/memory"
)

type fakeSubnetRepo struct {
	lists map[domain.ListType][]string
}

func (f *fakeSubnetRepo) GetSubnetLists(_ context.Context, listType domain.ListType) ([]string, error) {
	return f.lists[listType], nil
}

func (f *fakeSubnetRepo) SaveSubnetList(_ context.Context, listType domain.ListType, cidrs []string) error {
	f.lists[listType] = append([]string(nil), cidrs...)
	return nil
}

func (f *fakeSubnetRepo) ClearSubnetList(_ context.Context, listType domain.ListType) error {
	delete(f.lists, listType)
	return nil
}

func (f *fakeSubnetRepo) AddCIDRToSubnetList(_ context.Context, listType domain.ListType, cidr string) error {
	f.lists[listType] = append(f.lists[listType], cidr)
	return nil
}

func (f *fakeSubnetRepo) RemoveCIDRFromSubnetList(_ context.Context, listType domain.ListType, cidr string) error {
	s := f.lists[listType]
	out := make([]string, 0, len(s))
	for _, c := range s {
		if c == cidr {
			continue
		}
		out = append(out, c)
	}
	f.lists[listType] = out
	return nil
}

func TestRateLimiterService_WhitelistAllows(t *testing.T) {
	repo := &fakeSubnetRepo{lists: make(map[domain.ListType][]string)}
	// add whitelist
	repo.lists[domain.Whitelist] = []string{"192.168.1.0/24"}

	limitRepo := mem.NewBucketsDB()
	cfg := &config.Limits{LoginAttempts: 1, PasswordAttempts: 1, IPAttempts: 1, Window: 1 * time.Second}

	svc, err := NewRateLimiterService(repo, limitRepo, cfg)
	if err != nil {
		t.Fatalf("NewRateLimiterService: %v", err)
	}
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	ok, err := svc.Check(context.Background(), "u", "p", net.ParseIP("192.168.1.5"))
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected allow due to whitelist")
	}
}

func TestRateLimiterService_PasswordLimitAndReset(t *testing.T) {
	repo := &fakeSubnetRepo{lists: make(map[domain.ListType][]string)}
	limitRepo := mem.NewBucketsDB()
	cfg := &config.Limits{LoginAttempts: 10, PasswordAttempts: 1, IPAttempts: 10, Window: 200 * time.Millisecond}

	svc, err := NewRateLimiterService(repo, limitRepo, cfg)
	if err != nil {
		t.Fatalf("NewRateLimiterService: %v", err)
	}
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	ip := net.ParseIP("10.0.0.1")
	// first attempt allowed
	ok, err := svc.Check(context.Background(), "user", "secret", ip)
	if err != nil || !ok {
		t.Fatalf("expected first attempt allowed, got ok=%v err=%v", ok, err)
	}

	// second attempt with same password should be denied (PasswordAttempts=1)
	ok, err = svc.Check(context.Background(), "user2", "secret", ip)
	if err != nil {
		t.Fatalf("unexpected error on second attempt: %v", err)
	}
	if ok {
		t.Fatalf("expected second attempt to be denied due to password limit")
	}

	// reset password counter and expect allow
	if err := svc.Reset(context.Background(), "user2", "secret", ip); err != nil {
		t.Fatalf("Reset error: %v", err)
	}
	ok, err = svc.Check(context.Background(), "user2", "secret", ip)
	if err != nil || !ok {
		t.Fatalf("expected allowed after reset, got ok=%v err=%v", ok, err)
	}
}

func TestRateLimiterService_BlacklistDenies(t *testing.T) {
	repo := &fakeSubnetRepo{lists: make(map[domain.ListType][]string)}
	repo.lists[domain.Blacklist] = []string{"8.8.8.0/24"}
	limitRepo := mem.NewBucketsDB()
	cfg := &config.Limits{LoginAttempts: 10, PasswordAttempts: 10, IPAttempts: 10, Window: 1 * time.Second}

	svc, err := NewRateLimiterService(repo, limitRepo, cfg)
	if err != nil {
		t.Fatalf("NewRateLimiterService: %v", err)
	}
	if err := svc.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	ok, err := svc.Check(context.Background(), "u", "p", net.ParseIP("8.8.8.8"))
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected deny due to blacklist")
	}
}
