package abfclient

import (
	"context"
	"fmt"

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/api/proto/anti_bruteforce/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pbv1.AntiBruteforceClient
}

// New создает новый gRPC-клиент для AntiBruteforce-сервиса.
//
// address — например, "localhost:50051".
func New(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("abfclient: dial %s: %w", address, err)
	}

	return &Client{
		conn:   conn,
		client: pbv1.NewAntiBruteforceClient(conn),
	}, nil
}

// Close закрывает gRPC-соединение.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Check вызывает RPC Check.
func (c *Client) Check(ctx context.Context, login, password, ip string) (bool, error) {
	resp, err := c.client.CheckAttempt(ctx, &pbv1.CheckAttemptRequest{
		Login:    login,
		Password: password,
		Ip:       ip,
	})
	if err != nil {
		return false, err
	}
	return resp.GetOk(), nil
}

// ResetBucket вызывает RPC ResetBucket.
func (c *Client) ResetBucket(ctx context.Context, login, ip string) error {
	_, err := c.client.ResetBucket(ctx, &pbv1.ResetBucketRequest{
		Login: login,
		Ip:    ip,
	})
	return err
}

// AddToWhitelist вызывает RPC AddToWhitelist.
func (c *Client) AddToWhitelist(ctx context.Context, cidr string) error {
	_, err := c.client.AddToWhitelist(ctx, &pbv1.ManageCIDRRequest{Cidr: cidr})
	return err
}

// RemoveFromWhitelist вызывает RPC RemoveFromWhitelist.
func (c *Client) RemoveFromWhitelist(ctx context.Context, cidr string) error {
	_, err := c.client.RemoveFromWhitelist(ctx, &pbv1.ManageCIDRRequest{Cidr: cidr})
	return err
}

// AddToBlacklist вызывает RPC AddToBlacklist.
func (c *Client) AddToBlacklist(ctx context.Context, cidr string) error {
	_, err := c.client.AddToBlacklist(ctx, &pbv1.ManageCIDRRequest{Cidr: cidr})
	return err
}

// RemoveFromBlacklist вызывает RPC RemoveFromBlacklist.
func (c *Client) RemoveFromBlacklist(ctx context.Context, cidr string) error {
	_, err := c.client.RemoveFromBlacklist(ctx, &pbv1.ManageCIDRRequest{Cidr: cidr})
	return err
}
