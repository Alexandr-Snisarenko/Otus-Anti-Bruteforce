package subnetlist

import (
	"context"
	"fmt"
	"net"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ports"
)

var _ ports.SubnetPolicer = (*SubnetPolicer)(nil)

type SubnetPolicer struct {
	Whitelist *SubnetList
	Blacklist *SubnetList
	repo      ports.SubnetRepo
}

func NewSubnetPolicer(repo ports.SubnetRepo) (*SubnetPolicer, error) {
	return &SubnetPolicer{
		Whitelist: NewSubnetList(domain.Whitelist),
		Blacklist: NewSubnetList(domain.Blacklist),
		repo:      repo,
	}, nil
}

func (sp *SubnetPolicer) Check(ip net.IP) domain.PolicyDecision {
	if sp.Blacklist != nil && sp.Blacklist.Contains(ip) {
		return domain.DecisionDeny
	}

	if sp.Whitelist != nil && sp.Whitelist.Contains(ip) {
		return domain.DecisionAllow
	}

	return domain.DecisionContinue
}

func (sp *SubnetPolicer) ReloadSubnets(ctx context.Context) error {
	if sp.Whitelist != nil {
		if err := sp.Whitelist.Load(ctx, sp.repo); err != nil {
			return fmt.Errorf("update whitelist: %w", err)
		}
	}

	if sp.Blacklist != nil {
		if err := sp.Blacklist.Load(ctx, sp.repo); err != nil {
			return fmt.Errorf("update blacklist: %w", err)
		}
	}

	return nil
}
