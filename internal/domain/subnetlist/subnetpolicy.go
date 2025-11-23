package subnetlist

import (
	"context"
	"fmt"
	"net"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
)

type SubnetPolicy struct {
	Whitelist *SubnetList
	Blacklist *SubnetList
}

func NewSubnetPolicy(ctx context.Context, repo ports.SubnetRepo) (*SubnetPolicy, error) {
	whitelist := NewSubnetList(domain.Whitelist)
	if err := whitelist.Load(ctx, repo); err != nil {
		return nil, fmt.Errorf("load whitelist: %w", err)
	}

	blacklist := NewSubnetList(domain.Blacklist)
	if err := blacklist.Load(ctx, repo); err != nil {
		return nil, fmt.Errorf("load blacklist: %w", err)
	}

	return &SubnetPolicy{
		Whitelist: whitelist,
		Blacklist: blacklist,
	}, nil
}

func (sp *SubnetPolicy) Check(ip net.IP) domain.PolicyDecision {
	if sp.Blacklist != nil && sp.Blacklist.Contains(ip) {
		return domain.DecisionDeny
	}

	if sp.Whitelist != nil && sp.Whitelist.Contains(ip) {
		return domain.DecisionAllow
	}

	return domain.DecisionContinue
}
