package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
)

type fakeRepo struct {
	addCalled bool
	remCalled bool
	lastType  domain.ListType
	lastCIDR  string
	err       error
}

func (f *fakeRepo) GetSubnetLists(_ context.Context, _ domain.ListType) ([]string, error) {
	return nil, nil
}

func (f *fakeRepo) SaveSubnetList(_ context.Context, _ domain.ListType, _ []string) error {
	return nil
}

func (f *fakeRepo) ClearSubnetList(_ context.Context, _ domain.ListType) error { return nil }

func (f *fakeRepo) AddCIDRToSubnetList(_ context.Context, listType domain.ListType, cidr string) error {
	f.addCalled = true
	f.lastType = listType
	f.lastCIDR = cidr
	return f.err
}

func (f *fakeRepo) RemoveCIDRFromSubnetList(_ context.Context, listType domain.ListType, cidr string) error {
	f.remCalled = true
	f.lastType = listType
	f.lastCIDR = cidr
	return f.err
}

type fakePublisher struct {
	called bool
	err    error
}

func (f *fakePublisher) PublishSubnetUpdated(_ context.Context) error {
	f.called = true
	return f.err
}

func TestSubnetListService_AddRemove(t *testing.T) {
	tests := []struct {
		name                string
		call                func(s *SubnetListService) error
		wantRepoCalled      bool
		wantPublisherCalled bool
	}{
		{"AddToWhitelist", func(s *SubnetListService) error {
			return s.AddToWhitelist(context.Background(), "1.2.3.0/24")
		}, true, true},
		{"AddToBlacklist", func(s *SubnetListService) error {
			return s.AddToBlacklist(context.Background(), "1.2.3.0/24")
		}, true, true},
		{"RemoveFromWhitelist", func(s *SubnetListService) error {
			return s.RemoveFromWhitelist(context.Background(), "1.2.3.0/24")
		}, true, true},
		{"RemoveFromBlacklist", func(s *SubnetListService) error {
			return s.RemoveFromBlacklist(context.Background(), "1.2.3.0/24")
		}, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			pub := &fakePublisher{}
			s := NewSubnetListService(repo, pub)

			if err := tc.call(s); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.addCalled == false && repo.remCalled == false && tc.wantRepoCalled {
				t.Fatalf("expected repo to be called for %s", tc.name)
			}
			if !pub.called && tc.wantPublisherCalled {
				t.Fatalf("expected publisher to be called for %s", tc.name)
			}
		})
	}
}

func TestSubnetListService_Errors(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		repo := &fakeRepo{err: errors.New("repo fail")}
		pub := &fakePublisher{}
		s := NewSubnetListService(repo, pub)
		if err := s.AddToWhitelist(context.Background(), "1.2.3.0/24"); err == nil {
			t.Fatalf("expected error when repo fails")
		}
		if pub.called {
			t.Fatalf("publisher should not be called when repo fails")
		}
	})

	t.Run("publisher error", func(t *testing.T) {
		repo := &fakeRepo{}
		pub := &fakePublisher{err: errors.New("pub fail")}
		s := NewSubnetListService(repo, pub)
		if err := s.AddToWhitelist(context.Background(), "1.2.3.0/24"); err == nil {
			t.Fatalf("expected error when publisher fails")
		}
		if !repo.addCalled {
			t.Fatalf("repo should have been called before publisher")
		}
	})
}
