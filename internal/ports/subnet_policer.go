package ports

import (
	"context"
	"net"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
)

// SubnetHolder — интерфейс для управления подсетями.
type SubnetHolder interface {
	ReloadSubnets(ctx context.Context) error
}

// SubnetPolicer — интерфейс для проверки IP по подсетям.
type SubnetPolicer interface {
	Check(ip net.IP) domain.PolicyDecision
	SubnetHolder
}
