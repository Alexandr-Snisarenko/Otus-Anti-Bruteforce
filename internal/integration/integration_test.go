//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/abfclient"
	"github.com/stretchr/testify/require"
)

const (
	project    = "abf-it"
	composeYml = "../../docker-compose-it.yml"
	grpcAddr   = "127.0.0.1:50051"
)

// These values mirror defaults from configs/config.yml used by the service
const (
	loginAttempts    = 5
	passwordAttempts = 10
	ipAttempts       = 20
)

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	must(ctx, "docker", "compose", "-f", composeYml, "-p", project, "up", "-d", "--build")
	mustWaitTCP(ctx, grpcAddr, 90*time.Second)
	mustWaitABFReady(grpcAddr, 90*time.Second)

	code := m.Run()

	if code != 0 {
		_ = exec.Command("docker", "compose", "-f", composeYml, "-p", project, "logs").Run()
	}

	_ = exec.CommandContext(context.Background(),
		"docker", "compose", "-f", composeYml, "-p", project, "down", "-v",
	).Run()

	os.Exit(code)
}

func Test_ABF_AllAPI_Methods(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cl, err := abfclient.New(grpcAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cl.Close() })

	require.NoError(t, cl.AddToWhitelist(ctx, "192.168.2.0/24"))

	ok, err := cl.Check(ctx, "user-wl", "secret-wl", "192.168.2.10")
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, cl.RemoveFromWhitelist(ctx, "192.168.2.0/24"))

	require.NoError(t, cl.AddToBlacklist(ctx, "10.0.0.0/8"))

	require.Eventually(t, func() bool {
		ok, err := cl.Check(ctx, "user-bl", "secret-bl", "10.1.2.3")
		return err == nil && ok == false
	}, 3*time.Second, 200*time.Millisecond)

	require.NoError(t, cl.RemoveFromBlacklist(ctx, "10.0.0.0/8"))

	require.NoError(t, cl.ResetBucket(ctx, "user-reset", "11.22.33.44"))
}

func Test_ABF_RateLimits_Login_Password_IP_And_Reset(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cl, err := abfclient.New(grpcAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cl.Close() })

	// 1) Login limit
	login := "integration-login-limit"
	for i := 0; i < loginAttempts; i++ {
		ok, err := cl.Check(ctx, login, fmt.Sprintf("pw-%d", i), fmt.Sprintf("10.11.%d.%d", i/256, i%256))
		require.NoError(t, err)
		require.True(t, ok)
	}

	ok, err := cl.Check(ctx, login, "pw-final", "9.9.9.9")
	require.NoError(t, err)
	require.False(t, ok, "expected login limit to deny further attempts")

	// Reset by login (ResetBucket expects login and ip)
	require.NoError(t, cl.ResetBucket(ctx, login, "9.9.9.9"))
	ok, err = cl.Check(ctx, login, "pw-after-reset", "9.9.9.9")
	require.NoError(t, err)
	require.True(t, ok, "expected login allowed after reset")

	// 2) Password limit (same password, different logins/ips)
	password := "integration-password-limit"
	for i := 0; i < passwordAttempts; i++ {
		l := fmt.Sprintf("user-pw-%d", i)
		ip := fmt.Sprintf("20.21.%d.%d", i/256, i%256)
		ok, err := cl.Check(ctx, l, password, ip)
		require.NoError(t, err)
		require.True(t, ok)
	}

	ok, err = cl.Check(ctx, "some-other-user", password, "20.0.0.1")
	require.NoError(t, err)
	require.False(t, ok, "expected password limit to deny further attempts")

	// 3) IP limit
	ip := "123.123.123.1"
	for i := 0; i < ipAttempts; i++ {
		ok, err := cl.Check(ctx, fmt.Sprintf("user-ip-%d", i), fmt.Sprintf("pw-ip-%d", i), ip)
		require.NoError(t, err)
		require.True(t, ok)
	}

	ok, err = cl.Check(ctx, "another-user", "another-pw", ip)
	require.NoError(t, err)
	require.False(t, ok, "expected IP limit to deny further attempts")

	// Reset by IP
	require.NoError(t, cl.ResetBucket(ctx, "another-user", ip))
	ok, err = cl.Check(ctx, "another-user", "another-pw", ip)
	require.NoError(t, err)
	require.True(t, ok, "expected IP allowed after reset")
}

func must(ctx context.Context, name string, args ...string) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Errorf("command failed: %s %v: %w", name, args, err))
	}
}

func mustWaitTCP(ctx context.Context, address string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		d := net.Dialer{Timeout: 500 * time.Millisecond}
		c, err := d.DialContext(ctx, "tcp", address)
		if err == nil {
			_ = c.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	panic("timeout waiting for tcp " + address)
}

func mustWaitABFReady(addr string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		c, err := abfclient.New(grpcAddr)
		if err == nil {
			_, err = c.Check(ctx, "user", "secret", "1.2.3.4")

			c.Close()
			cancel()

			// Для "готовности" нам важно не "ok", а что сервер уже отвечает (даже ошибкой домена).
			if err == nil {
				return
			}

		}

		time.Sleep(400 * time.Millisecond)
	}

	panic("timeout waiting for abf grpc ready " + addr)
}
