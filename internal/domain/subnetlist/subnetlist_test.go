package subnetlist

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/storage/memory"
)

func TestIsIPInList_Table(t *testing.T) {
	tests := []struct {
		name  string
		CIDRs []string
		ip    string
		want  bool
		err   error
	}{
		{
			name:  "ipv4 found",
			CIDRs: []string{"192.168.0.0/24"},

			ip:   "192.168.0.10",
			want: true,
			err:  nil,
		},
		{
			name:  "ipv4 not found",
			CIDRs: []string{"10.0.0.0/8"},
			ip:    "192.168.1.1",
			want:  false,
			err:   nil,
		},
		{
			name:  "empty list",
			CIDRs: []string{},
			ip:    "1.2.3.4",
			want:  false,
			err:   nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list := NewSubnetList(domain.Whitelist)
			for _, cidr := range tc.CIDRs {
				err := list.Add(cidr)
				if err != nil && !errors.Is(err, tc.err) {
					t.Fatalf("unexpected error adding cidr %s: %v", cidr, err)
				}
			}
			got := list.Contains(net.ParseIP(tc.ip))
			if got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestAdd_InvalidCIDR(t *testing.T) {
	tests := []struct {
		name string
		CIDR string
	}{
		{
			name: "not-an-CIDR",
			CIDR: "invalid-cidr",
		},
		{
			name: "missing-slash",
			CIDR: "192.168.0.0",
		},
		{
			name: "not-valid-CIDR",
			CIDR: "10.10.10.10/24",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list := NewSubnetList(domain.Whitelist)
			err := list.Add(tc.CIDR)
			if err == nil {
				t.Fatalf("expected error for invalid CIDR, got nil")
			}
			if !errors.Is(err, ErrInvalidCIDR) {
				t.Fatalf("expected ErrInvalidCIDR, got %v", err)
			}
		})
	}
}

// Тесты загрузки списка подсетей из SubnetRepo с использованием memory.SubnetListDB.
func TestLoadFromSubnetRepo_Success(t *testing.T) {
	repo := memory.NewSubnetListDB()

	// подготовим данные в хранилище
	if err := repo.SaveSubnetList(context.Background(), domain.Whitelist,
		[]string{"192.168.10.0/24", "10.0.0.0/8"}); err != nil {
		t.Fatalf("SaveSubnetList error: %v", err)
	}

	list := NewSubnetList(domain.Whitelist)
	if err := list.Load(context.Background(), repo); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// проверим, что IP входит
	ok := list.Contains(net.ParseIP("192.168.10.5"))
	if !ok {
		t.Fatalf("expected IP to be contained after load")
	}

	// и что другой IP не входит
	ok = list.Contains(net.ParseIP("172.16.0.1"))
	if ok {
		t.Fatalf("expected IP not to be contained after load")
	}
}

func TestLoadFromSubnetRepo_InvalidCIDR(t *testing.T) {
	repo := memory.NewSubnetListDB()

	// сохраним невалидный CIDR
	if err := repo.SaveSubnetList(context.Background(), domain.Whitelist, []string{"not-a-cidr"}); err != nil {
		t.Fatalf("SaveSubnetList error: %v", err)
	}

	list := NewSubnetList(domain.Whitelist)
	err := list.Load(context.Background(), repo)
	if err == nil {
		t.Fatalf("expected error when loading invalid CIDR, got nil")
	}
	if !errors.Is(err, ErrInvalidCIDR) {
		t.Fatalf("expected ErrInvalidCIDR, got %v", err)
	}
}
