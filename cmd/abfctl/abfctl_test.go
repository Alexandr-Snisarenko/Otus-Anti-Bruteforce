package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"unsafe"

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/api/proto/anti_bruteforce/v1"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/abfclient"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
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

func TestCheckCmd_PrintAllowedAndNotAllowed(t *testing.T) {
	tests := []struct {
		name   string
		respOk bool
		want   string
	}{
		{"allowed", true, "Request is allowed"},
		{"not_allowed", false, "Request is not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newCheckCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)

			_ = cmd.Flags().Set("login", "user")
			_ = cmd.Flags().Set("pass", "secret")
			_ = cmd.Flags().Set("ip", "127.0.0.1")

			fake := &fakePBClient{checkResp: &pbv1.CheckAttemptResponse{Ok: tt.respOk}}
			c := &abfclient.Client{}
			setPBClient(c, fake)
			cmd.SetContext(context.WithValue(context.Background(), clientKey, c))

			if err := cmd.RunE(cmd, nil); err != nil {
				t.Fatalf("RunE returned error: %v", err)
			}

			if got := out.String(); !bytes.Contains([]byte(got), []byte(tt.want)) {
				t.Fatalf("unexpected output: %q, want contains %q", got, tt.want)
			}
		})
	}
}

func TestCheckCmd_ErrorFromClient(t *testing.T) {
	cmd := newCheckCmd()
	_ = cmd.Flags().Set("login", "user")
	_ = cmd.Flags().Set("pass", "secret")
	_ = cmd.Flags().Set("ip", "127.0.0.1")

	fake := &fakePBClient{checkErr: errors.New("rpc failed")}
	c := &abfclient.Client{}
	setPBClient(c, fake)
	cmd.SetContext(context.WithValue(context.Background(), clientKey, c))

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestResetCmd_SuccessAndError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cmd := newResetCmd()
		_ = cmd.Flags().Set("login", "user")
		_ = cmd.Flags().Set("ip", "1.2.3.4")

		fake := &fakePBClient{resetErr: nil}
		c := &abfclient.Client{}
		setPBClient(c, fake)
		cmd.SetContext(context.WithValue(context.Background(), clientKey, c))

		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.lastReset.GetLogin() != "user" || fake.lastReset.GetIp() != "1.2.3.4" {
			t.Fatalf("unexpected args in ResetBucket: %+v", fake.lastReset)
		}
	})

	t.Run("error", func(t *testing.T) {
		cmd := newResetCmd()
		_ = cmd.Flags().Set("login", "user")
		_ = cmd.Flags().Set("ip", "1.2.3.4")

		fake := &fakePBClient{resetErr: errors.New("fail")}
		c := &abfclient.Client{}
		setPBClient(c, fake)
		cmd.SetContext(context.WithValue(context.Background(), clientKey, c))

		if err := cmd.RunE(cmd, nil); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestList_AddRemove(t *testing.T) {
	tests := []struct {
		name     string
		add      bool
		wantCidr string
		wantErr  bool
	}{
		{"add_whitelist", true, "10.0.0.0/24", false},
		{"remove_whitelist", false, "10.0.1.0/24", false},
		{"add_blacklist", true, "192.168.0.0/24", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := listSpec{
				name: "testlist",
				addFn: func(cmd *cobra.Command, cidr string) error {
					return getClient(cmd).AddToWhitelist(cmd.Context(), cidr)
				},
				removeFn: func(cmd *cobra.Command, cidr string) error {
					return getClient(cmd).RemoveFromWhitelist(cmd.Context(), cidr)
				},
			}

			var cmd *cobra.Command
			if tt.add {
				cmd = newListAddCmd(spec)
			} else {
				cmd = newListRemoveCmd(spec)
			}

			_ = cmd.Flags().Set("cidr", tt.wantCidr)
			fake := &fakePBClient{cidrErr: nil}
			c := &abfclient.Client{}
			setPBClient(c, fake)
			cmd.SetContext(context.WithValue(context.Background(), clientKey, c))

			if err := cmd.RunE(cmd, nil); (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}
			if fake.lastCidr != tt.wantCidr {
				t.Fatalf("expected cidr %q, got %q", tt.wantCidr, fake.lastCidr)
			}
		})
	}
}

// setPBClient sets the unexported pb client field on abfclient.Client using
// reflection. This is necessary for tests to inject a fake implementation.
func setPBClient(c *abfclient.Client, pb pbv1.AntiBruteforceClient) {
	v := reflect.ValueOf(c).Elem()
	f := v.FieldByName("client")
	// make unexported field settable
	fv := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	fv.Set(reflect.ValueOf(pb))
}
