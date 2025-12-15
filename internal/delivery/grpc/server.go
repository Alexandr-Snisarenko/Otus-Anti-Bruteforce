package grpcserver

import (
	"context"
	"net"

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/api/proto/anti_bruteforce/v1"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/app"
)

var _ pbv1.AntiBruteforceServer = (*Server)(nil)

type Server struct {
	pbv1.UnimplementedAntiBruteforceServer
	rateLimiter app.RateLimiterUseCase
	subnetList  app.SubnetListUseCase
}

func NewServer(rateLimiter app.RateLimiterUseCase, subnetList app.SubnetListUseCase) *Server {
	return &Server{
		rateLimiter: rateLimiter,
		subnetList:  subnetList,
	}
}

func (s *Server) Check(
	ctx context.Context,
	req *pbv1.CheckAttemptRequest,
) (resp *pbv1.CheckAttemptResponse, err error) {
	if s.rateLimiter == nil {
		return nil, ErrRateLimiterNotConfigured
	}

	IP := net.ParseIP(req.Ip)
	if IP == nil {
		return nil, ErrInvalidIP
	}

	allowed, err := s.rateLimiter.Check(ctx, req.Login, req.Password, IP)
	if err != nil {
		return nil, err
	}
	return &pbv1.CheckAttemptResponse{Ok: allowed}, nil
}

func (s *Server) Reset(ctx context.Context, req *pbv1.ResetBucketRequest) (resp *pbv1.ResetBucketResponse, err error) {
	if s.rateLimiter == nil {
		return nil, ErrRateLimiterNotConfigured
	}

	IP := net.ParseIP(req.Ip)
	if IP == nil {
		return nil, ErrInvalidIP
	}

	if err := s.rateLimiter.Reset(ctx, req.Login, req.Password, IP); err != nil {
		return nil, err
	}
	return &pbv1.ResetBucketResponse{}, nil
}

func (s *Server) AddToWhitelist(
	ctx context.Context,
	req *pbv1.ManageCIDRRequest,
) (resp *pbv1.ManageCIDRResponse, err error) {
	if s.subnetList == nil {
		return nil, ErrSubnetListNotConfigured
	}

	if err := s.subnetList.AddToWhitelist(ctx, req.Cidr); err != nil {
		return nil, err
	}
	return &pbv1.ManageCIDRResponse{}, nil
}

func (s *Server) AddToBlacklist(
	ctx context.Context,
	req *pbv1.ManageCIDRRequest,
) (resp *pbv1.ManageCIDRResponse, err error) {
	if s.subnetList == nil {
		return nil, ErrSubnetListNotConfigured
	}

	if err := s.subnetList.AddToBlacklist(ctx, req.Cidr); err != nil {
		return nil, err
	}
	return &pbv1.ManageCIDRResponse{}, nil
}

func (s *Server) RemoveFromWhitelist(
	ctx context.Context,
	req *pbv1.ManageCIDRRequest,
) (resp *pbv1.ManageCIDRResponse, err error) {
	if s.subnetList == nil {
		return nil, ErrSubnetListNotConfigured
	}

	if err := s.subnetList.RemoveFromWhitelist(ctx, req.Cidr); err != nil {
		return nil, err
	}
	return &pbv1.ManageCIDRResponse{}, nil
}

func (s *Server) RemoveFromBlacklist(
	ctx context.Context,
	req *pbv1.ManageCIDRRequest,
) (resp *pbv1.ManageCIDRResponse, err error) {
	if s.subnetList == nil {
		return nil, ErrSubnetListNotConfigured
	}

	if err := s.subnetList.RemoveFromBlacklist(ctx, req.Cidr); err != nil {
		return nil, err
	}
	return &pbv1.ManageCIDRResponse{}, nil
}
