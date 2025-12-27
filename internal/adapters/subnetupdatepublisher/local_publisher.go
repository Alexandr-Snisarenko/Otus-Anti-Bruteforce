package subnetupdatepublisher

import (
	"context"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ports"
)

type LocalSubnetUpdatesPublisher struct {
	holder ports.SubnetHolder
}

func (p *LocalSubnetUpdatesPublisher) PublishSubnetUpdated(ctx context.Context) error {
	return p.holder.ReloadSubnets(ctx)
}

func NewLocalSubnetUpdatesPublisher(
	holder ports.SubnetHolder,
) *LocalSubnetUpdatesPublisher {
	return &LocalSubnetUpdatesPublisher{
		holder: holder,
	}
}
