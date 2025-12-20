package grpcserver

import (
	"context"
	"errors"
	"net"
	"testing"

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/api/proto/anti_bruteforce/v1"
)

type fakeRateLimiter struct {
	checkFn func(ctx context.Context, login, password string, ip net.IP) (bool, error)
	resetFn func(ctx context.Context, login, password string, ip net.IP) error
}

func (f *fakeRateLimiter) Check(ctx context.Context, login, password string, ip net.IP) (bool, error) {
	return f.checkFn(ctx, login, password, ip)
}

func (f *fakeRateLimiter) Reset(ctx context.Context, login, password string, ip net.IP) error {
	return f.resetFn(ctx, login, password, ip)
}

type fakeSubnetList struct {
	addWFn func(ctx context.Context, cidr string) error
	addBFn func(ctx context.Context, cidr string) error
	remWFn func(ctx context.Context, cidr string) error
	remBFn func(ctx context.Context, cidr string) error
}

func (f *fakeSubnetList) AddToWhitelist(ctx context.Context, cidr string) error {
	return f.addWFn(ctx, cidr)
}

func (f *fakeSubnetList) AddToBlacklist(ctx context.Context, cidr string) error {
	return f.addBFn(ctx, cidr)
}

func (f *fakeSubnetList) RemoveFromWhitelist(ctx context.Context, cidr string) error {
	return f.remWFn(ctx, cidr)
}

func (f *fakeSubnetList) RemoveFromBlacklist(ctx context.Context, cidr string) error {
	return f.remBFn(ctx, cidr)
}

func TestCheckAttempt_InvalidIP(t *testing.T) {
	s := NewServer(&fakeRateLimiter{}, &fakeSubnetList{})
	req := &pbv1.CheckAttemptRequest{Login: "u", Password: "p", Ip: "not-an-ip"}

	_, err := s.CheckAttempt(context.Background(), req)
	if !errors.Is(err, ErrInvalidIP) {
		t.Fatalf("expected ErrInvalidIP, got %v", err)
	}
}

func TestCheckAttempt_NotConfigured(t *testing.T) {
	s := NewServer(nil, &fakeSubnetList{})
	req := &pbv1.CheckAttemptRequest{Login: "u", Password: "p", Ip: "127.0.0.1"}

	_, err := s.CheckAttempt(context.Background(), req)
	if !errors.Is(err, ErrRateLimiterNotConfigured) {
		t.Fatalf("expected ErrRateLimiterNotConfigured, got %v", err)
	}
}

func TestCheckAttempt_Scenarios(t *testing.T) {
	cases := []struct {
		name    string
		checkFn func(context.Context, string, string, net.IP) (bool, error)
		wantOk  bool
		wantErr error
	}{
		{"allow", func(_ context.Context, _, _ string, _ net.IP) (bool, error) { return true, nil }, true, nil},
		{"deny", func(_ context.Context, _, _ string, _ net.IP) (bool, error) { return false, nil }, false, nil},
		{"error", func(_ context.Context, _, _ string, _ net.IP) (bool, error) {
			return false, errors.New("some error")
		}, false, errors.New("some error")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := &fakeRateLimiter{
				checkFn: tc.checkFn, resetFn: func(_ context.Context, _, _ string, _ net.IP) error { return nil },
			}
			s := NewServer(fr, &fakeSubnetList{})
			req := &pbv1.CheckAttemptRequest{Login: "u", Password: "p", Ip: "127.0.0.1"}
			resp, err := s.CheckAttempt(context.Background(), req)
			if tc.wantErr != nil {
				if err == nil || err.Error() != tc.wantErr.Error() {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatalf("expected response, got nil")
			}
			if resp.Ok != tc.wantOk {
				t.Fatalf("want ok=%v, got %v", tc.wantOk, resp.Ok)
			}
		})
	}
}

func TestResetBucket(t *testing.T) {
	called := false
	fr := &fakeRateLimiter{
		checkFn: func(_ context.Context, _, _ string, _ net.IP) (bool, error) { return true, nil },
		resetFn: func(_ context.Context, _, _ string, _ net.IP) error { called = true; return nil },
	}
	s := NewServer(fr, &fakeSubnetList{})
	req := &pbv1.ResetBucketRequest{Login: "u", Password: "p", Ip: "127.0.0.1"}
	_, err := s.ResetBucket(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected Reset to be called")
	}

	// invalid ip
	_, err = s.ResetBucket(context.Background(), &pbv1.ResetBucketRequest{Login: "u", Password: "p", Ip: "bad"})
	if !errors.Is(err, ErrInvalidIP) {
		t.Fatalf("expected ErrInvalidIP, got %v", err)
	}
}

func TestSubnetListMethods(t *testing.T) {
	var calledAddW, calledAddB, calledRemW, calledRemB bool
	fs := &fakeSubnetList{
		addWFn: func(_ context.Context, _ string) error { calledAddW = true; return nil },
		addBFn: func(_ context.Context, _ string) error { calledAddB = true; return nil },
		remWFn: func(_ context.Context, _ string) error { calledRemW = true; return nil },
		remBFn: func(_ context.Context, _ string) error { calledRemB = true; return nil },
	}
	s := NewServer(&fakeRateLimiter{}, fs)

	if _, err := s.AddToWhitelist(context.Background(), &pbv1.ManageCIDRRequest{Cidr: "1.2.3.0/24"}); err != nil {
		t.Fatalf("AddToWhitelist error: %v", err)
	}
	if !calledAddW {
		t.Fatalf("AddToWhitelist not forwarded")
	}

	if _, err := s.AddToBlacklist(context.Background(), &pbv1.ManageCIDRRequest{Cidr: "1.2.3.0/24"}); err != nil {
		t.Fatalf("AddToBlacklist error: %v", err)
	}
	if !calledAddB {
		t.Fatalf("AddToBlacklist not forwarded")
	}

	if _, err := s.RemoveFromWhitelist(context.Background(), &pbv1.ManageCIDRRequest{Cidr: "1.2.3.0/24"}); err != nil {
		t.Fatalf("RemoveFromWhitelist error: %v", err)
	}
	if !calledRemW {
		t.Fatalf("RemoveFromWhitelist not forwarded")
	}

	if _, err := s.RemoveFromBlacklist(context.Background(), &pbv1.ManageCIDRRequest{Cidr: "1.2.3.0/24"}); err != nil {
		t.Fatalf("RemoveFromBlacklist error: %v", err)
	}
	if !calledRemB {
		t.Fatalf("RemoveFromBlacklist not forwarded")
	}

	// nil subnetList
	s2 := NewServer(&fakeRateLimiter{}, nil)
	if _, err := s2.AddToWhitelist(
		context.Background(), &pbv1.ManageCIDRRequest{Cidr: "a"}); !errors.Is(err, ErrSubnetListNotConfigured) {
		t.Fatalf("expected ErrSubnetListNotConfigured, got %v", err)
	}
}
