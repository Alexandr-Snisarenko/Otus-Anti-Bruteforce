package subnetupdatepublisher

import (
	"context"
	"errors"
	"testing"
)

type fakeHolder struct {
	called bool
	err    error
}

func (f *fakeHolder) ReloadSubnets(_ context.Context) error {
	f.called = true
	return f.err
}

func TestLocalPublish_SuccessAndError(t *testing.T) {
	fh := &fakeHolder{}
	p := NewLocalSubnetUpdatesPublisher(fh)

	if err := p.PublishSubnetUpdated(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fh.called {
		t.Fatal("expected ReloadSubnets to be called")
	}

	// error propagation
	fh2 := &fakeHolder{err: errors.New("fail")}
	p2 := NewLocalSubnetUpdatesPublisher(fh2)
	if err := p2.PublishSubnetUpdated(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}
