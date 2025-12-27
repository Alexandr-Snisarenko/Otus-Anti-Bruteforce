package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func sliceToSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

func TestSaveAndGetSubnetList(t *testing.T) {
	db := NewSubnetListDB()
	ctx := context.Background()

	want := []string{"10.0.0.0/8", "192.168.1.0/24"}
	if err := db.SaveSubnetList(ctx, "whitelist", want); err != nil {
		t.Fatalf("SaveSubnetList error: %v", err)
	}

	got, err := db.GetSubnetLists(ctx, "whitelist")
	if err != nil {
		t.Fatalf("GetSubnetLists error: %v", err)
	}

	gm := sliceToSet(got)
	for _, w := range want {
		if _, ok := gm[w]; !ok {
			t.Fatalf("expected CIDR %s to be present, got %v", w, got)
		}
	}

	// non-existing list should return empty slice
	empty, err := db.GetSubnetLists(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("GetSubnetLists error on non-existing: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty slice for non-existing list, got %v", empty)
	}
}

func TestAddAndRemoveCIDR(t *testing.T) {
	db := NewSubnetListDB()
	ctx := context.Background()

	if err := db.AddCIDRToSubnetList(ctx, "blacklist", "8.8.8.8/32"); err != nil {
		t.Fatalf("AddCIDRToSubnetList error: %v", err)
	}

	got, _ := db.GetSubnetLists(ctx, "blacklist")
	if len(got) != 1 || got[0] != "8.8.8.8/32" {
		// order is not guaranteed, check as set
		gm := sliceToSet(got)
		if _, ok := gm["8.8.8.8/32"]; !ok {
			t.Fatalf("expected 8.8.8.8/32 in blacklist, got %v", got)
		}
	}

	if err := db.RemoveCIDRFromSubnetList(ctx, "blacklist", "8.8.8.8/32"); err != nil {
		t.Fatalf("RemoveCIDRFromSubnetList error: %v", err)
	}
	got, _ = db.GetSubnetLists(ctx, "blacklist")
	if len(got) != 0 {
		t.Fatalf("expected empty blacklist after remove, got %v", got)
	}
}

func TestClearSubnetList(t *testing.T) {
	db := NewSubnetListDB()
	ctx := context.Background()
	if err := db.SaveSubnetList(ctx, "wl", []string{"1.1.1.0/24"}); err != nil {
		t.Fatalf("SaveSubnetList error: %v", err)
	}
	if err := db.ClearSubnetList(ctx, "wl"); err != nil {
		t.Fatalf("ClearSubnetList error: %v", err)
	}
	got, _ := db.GetSubnetLists(ctx, "wl")
	if len(got) != 0 {
		t.Fatalf("expected empty list after clear, got %v", got)
	}
}

func TestConcurrentAdds(t *testing.T) {
	db := NewSubnetListDB()
	ctx := context.Background()

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			cidr := fmt.Sprintf("10.0.%d.0/24", i)
			_ = db.AddCIDRToSubnetList(ctx, "concurrent", cidr)
		}(i)
	}
	wg.Wait()

	got, _ := db.GetSubnetLists(ctx, "concurrent")
	if len(got) != n {
		// allow that string(i) produced odd bytes, but still verify no data race and reasonable count
		t.Fatalf("expected %d entries after concurrent adds, got %d", n, len(got))
	}
}
