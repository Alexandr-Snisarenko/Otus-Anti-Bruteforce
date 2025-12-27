package ports

import (
	"context"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
)

// SubnetRepo — абстракция для работы со списками подсетей в БД.
type SubnetRepo interface {
	GetSubnetLists(ctx context.Context, listType domain.ListType) ([]string, error)
	SaveSubnetList(ctx context.Context, listType domain.ListType, cidrs []string) error
	ClearSubnetList(ctx context.Context, listType domain.ListType) error
	AddCIDRToSubnetList(ctx context.Context, listType domain.ListType, cidr string) error
	RemoveCIDRFromSubnetList(ctx context.Context, listType domain.ListType, cidr string) error
}
