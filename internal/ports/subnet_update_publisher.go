package ports

import "context"

type SubnetUpdatesPublisher interface {
	PublishSubnetUpdated(ctx context.Context) error
}
