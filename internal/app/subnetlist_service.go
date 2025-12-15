package app

import (
	"context"
	"fmt"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
)

type SubnetListUseCase interface {
	AddToWhitelist(ctx context.Context, cidr string) error
	AddToBlacklist(ctx context.Context, cidr string) error
	RemoveFromWhitelist(ctx context.Context, cidr string) error
	RemoveFromBlacklist(ctx context.Context, cidr string) error
}

// Проверка реализации интерфейса SubnetListUseCase на этапе компиляции.
var _ SubnetListUseCase = (*SubnetListService)(nil)

type SubnetListService struct {
	subnetRepo ports.SubnetRepo
}

func NewSubnetListService(subnetRepo ports.SubnetRepo) *SubnetListService {
	return &SubnetListService{
		subnetRepo: subnetRepo,
	}
}

func (s *SubnetListService) AddToWhitelist(ctx context.Context, cidr string) error {
	if err := s.subnetRepo.AddCIDRToSubnetList(ctx, domain.Whitelist, cidr); err != nil {
		return fmt.Errorf("add to whitelist subnet: %w", err)
	}
	return nil
}

func (s *SubnetListService) AddToBlacklist(ctx context.Context, cidr string) error {
	if err := s.subnetRepo.AddCIDRToSubnetList(ctx, domain.Blacklist, cidr); err != nil {
		return fmt.Errorf("add to blacklist subnet: %w", err)
	}
	return nil
}

func (s *SubnetListService) RemoveFromWhitelist(ctx context.Context, cidr string) error {
	if err := s.subnetRepo.RemoveCIDRFromSubnetList(ctx, domain.Whitelist, cidr); err != nil {
		return fmt.Errorf("remove from whitelist subnet: %w", err)
	}
	return nil
}

func (s *SubnetListService) RemoveFromBlacklist(ctx context.Context, cidr string) error {
	if err := s.subnetRepo.RemoveCIDRFromSubnetList(ctx, domain.Blacklist, cidr); err != nil {
		return fmt.Errorf("remove from blacklist subnet: %w", err)
	}
	return nil
}
