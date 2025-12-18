package subnetlist

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	mem "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/memory"
)

func TestSubnetPolicer_Check_AllowDenyContinue(t *testing.T) {
	repo := mem.NewSubnetListDB()

	// whitelist contains 192.168.1.0/24
	if err := repo.SaveSubnetList(context.Background(), domain.Whitelist, []string{"192.168.1.0/24"}); err != nil {
		t.Fatalf("SaveSubnetList failed: %v", err)
	}
	// blacklist contains 10.0.0.0/8
	if err := repo.SaveSubnetList(context.Background(), domain.Blacklist, []string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("SaveSubnetList failed: %v", err)
	}

	sp, err := NewSubnetPolicer(repo)
	if err != nil {
		t.Fatalf("NewSubnetPolicer failed: %v", err)
	}

	if err := sp.ReloadSubnets(context.Background()); err != nil {
		t.Fatalf("ReloadSubnets failed: %v", err)
	}

	// IP in whitelist -> allow
	ip := net.ParseIP("192.168.1.5")
	if got := sp.Check(ip); got != domain.DecisionAllow {
		t.Fatalf("expected DecisionAllow for %s, got %v", ip, got)
	}

	// IP in blacklist -> deny
	ip2 := net.ParseIP("10.5.6.7")
	if got := sp.Check(ip2); got != domain.DecisionDeny {
		t.Fatalf("expected DecisionDeny for %s, got %v", ip2, got)
	}

	// IP in neither -> continue
	ip3 := net.ParseIP("8.8.8.8")
	if got := sp.Check(ip3); got != domain.DecisionContinue {
		t.Fatalf("expected DecisionContinue for %s, got %v", ip3, got)
	}
}

func TestSubnetPolicer_ReloadSubnets_InvalidCIDR(t *testing.T) {
	repo := mem.NewSubnetListDB()

	// Save invalid CIDR into whitelist
	if err := repo.SaveSubnetList(context.Background(), domain.Whitelist, []string{"bad-cidr"}); err != nil {
		t.Fatalf("SaveSubnetList failed: %v", err)
	}

	sp, err := NewSubnetPolicer(repo)
	if err != nil {
		t.Fatalf("NewSubnetPolicer failed: %v", err)
	}

	err = sp.ReloadSubnets(context.Background())
	if err == nil {
		t.Fatalf("expected error on ReloadSubnets with invalid CIDR, got nil")
	}
	if !errors.Is(err, ErrInvalidCIDR) {
		t.Fatalf("expected ErrInvalidCIDR wrapped, got %v", err)
	}
}
