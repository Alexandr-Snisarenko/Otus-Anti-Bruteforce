package abfclient

import (
	"context"
	"net"
	"reflect"
	"testing"
	"unsafe"

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/api/proto/anti_bruteforce/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type fakePBClient struct {
	checkResp *pbv1.CheckAttemptResponse
	checkErr  error
	resetErr  error
	cidrErr   error

	lastCheck *pbv1.CheckAttemptRequest
	lastReset *pbv1.ResetBucketRequest
	lastCidr  string
}

func (f *fakePBClient) CheckAttempt(
	_ context.Context,
	in *pbv1.CheckAttemptRequest,
	_ ...grpc.CallOption,
) (*pbv1.CheckAttemptResponse, error) {
	f.lastCheck = in
	return f.checkResp, f.checkErr
}

func (f *fakePBClient) ResetBucket(
	_ context.Context,
	in *pbv1.ResetBucketRequest,
	_ ...grpc.CallOption,
) (*pbv1.ResetBucketResponse, error) {
	f.lastReset = in
	return &pbv1.ResetBucketResponse{}, f.resetErr
}

func (f *fakePBClient) AddToWhitelist(
	_ context.Context,
	in *pbv1.ManageCIDRRequest,
	_ ...grpc.CallOption,
) (*pbv1.ManageCIDRResponse, error) {
	f.lastCidr = in.GetCidr()
	return &pbv1.ManageCIDRResponse{}, f.cidrErr
}

func (f *fakePBClient) RemoveFromWhitelist(
	_ context.Context,
	in *pbv1.ManageCIDRRequest,
	_ ...grpc.CallOption,
) (*pbv1.ManageCIDRResponse, error) {
	f.lastCidr = in.GetCidr()
	return &pbv1.ManageCIDRResponse{}, f.cidrErr
}

func (f *fakePBClient) AddToBlacklist(
	_ context.Context,
	in *pbv1.ManageCIDRRequest,
	_ ...grpc.CallOption,
) (*pbv1.ManageCIDRResponse, error) {
	f.lastCidr = in.GetCidr()
	return &pbv1.ManageCIDRResponse{}, f.cidrErr
}

func (f *fakePBClient) RemoveFromBlacklist(
	_ context.Context,
	in *pbv1.ManageCIDRRequest,
	_ ...grpc.CallOption,
) (*pbv1.ManageCIDRResponse, error) {
	f.lastCidr = in.GetCidr()
	return &pbv1.ManageCIDRResponse{}, f.cidrErr
}

func setPBClient(c *Client, pb pbv1.AntiBruteforceClient) {
	v := reflect.ValueOf(c).Elem()
	f := v.FieldByName("client")
	fv := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	fv.Set(reflect.ValueOf(pb))
}

func setConn(c *Client, conn *grpc.ClientConn) {
	v := reflect.ValueOf(c).Elem()
	f := v.FieldByName("conn")
	fv := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	fv.Set(reflect.ValueOf(conn))
}

func TestCheckAndResetAndCidrMethods_ForwardToPB(t *testing.T) {
	c := &Client{}
	fake := &fakePBClient{checkResp: &pbv1.CheckAttemptResponse{Ok: true}}
	setPBClient(c, fake)

	ok, err := c.Check(context.Background(), "user", "pass", "1.2.3.4")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}

	if !ok {
		t.Fatalf("expected ok=true, got false")
	}

	if fake.lastCheck.GetLogin() != "user" || fake.lastCheck.GetPassword() != "pass" ||
		fake.lastCheck.GetIp() != "1.2.3.4" {
		t.Fatalf("unexpected CheckAttempt args: %+v", fake.lastCheck)
	}

	// error forwarding
	fake.checkErr = status.Errorf(codes.Unavailable, "boom")
	if _, err := c.Check(context.Background(), "u", "p", "ip"); err == nil {
		t.Fatalf("expected error from Check, got nil")
	}

	// ResetBucket
	fake.resetErr = nil
	if err := c.ResetBucket(context.Background(), "user", "1.2.3.4"); err != nil {
		t.Fatalf("ResetBucket error: %v", err)
	}
	if fake.lastReset.GetLogin() != "user" || fake.lastReset.GetIp() != "1.2.3.4" {
		t.Fatalf("unexpected ResetBucket args: %+v", fake.lastReset)
	}

	fake.resetErr = status.Errorf(codes.Unavailable, "boom")
	if err := c.ResetBucket(context.Background(), "u", "ip"); err == nil {
		t.Fatalf("expected error from ResetBucket, got nil")
	}

	// CIDR methods
	fake.cidrErr = nil
	if err := c.AddToWhitelist(context.Background(), "10.0.0.0/24"); err != nil {
		t.Fatalf("AddToWhitelist error: %v", err)
	}
	if fake.lastCidr != "10.0.0.0/24" {
		t.Fatalf("unexpected cidr for add whitelist: %q", fake.lastCidr)
	}

	if err := c.RemoveFromWhitelist(context.Background(), "10.0.0.0/24"); err != nil {
		t.Fatalf("RemoveFromWhitelist error: %v", err)
	}

	if err := c.AddToBlacklist(context.Background(), "10.0.1.0/24"); err != nil {
		t.Fatalf("AddToBlacklist error: %v", err)
	}

	if err := c.RemoveFromBlacklist(context.Background(), "10.0.1.0/24"); err != nil {
		t.Fatalf("RemoveFromBlacklist error: %v", err)
	}
}

func TestClose_NilAndRealConn(t *testing.T) {
	// nil client/conn should not panic and should return nil
	c := &Client{}
	if err := c.Close(); err != nil {
		t.Fatalf("Close on empty client returned error: %v", err)
	}

	// start a real gRPC server and dial it, then inject conn and verify Close works
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	srv := grpc.NewServer()
	// register an empty server so dial succeeds
	pbv1.RegisterAntiBruteforceServer(srv, &pbv1.UnimplementedAntiBruteforceServer{})

	go srv.Serve(lis)
	defer srv.Stop()

	addr := lis.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	c2 := &Client{}
	setConn(c2, conn)
	setPBClient(c2, pbv1.NewAntiBruteforceClient(conn))

	if err := c2.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
