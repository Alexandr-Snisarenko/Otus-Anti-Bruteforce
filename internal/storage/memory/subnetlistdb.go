package memory

import (
	"context"
	"sync"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
)

// SubnetListDB — простая in-memory реализация для работы со списками подсетей.
// CIDR хранятся в map для быстрого поиска и удаления.
// Каждый тип списка подсетей хранится в отдельной map.
// Формат записи: map[listType]map[cidr]struct{}
// Например:
// data = {"whitelist": {"192.168.10.10/24": struct{}{}, "192.168.20.20/24": struct{}{}},}

type SubnetListDB struct {
	mu   sync.RWMutex
	data map[domain.ListType]map[string]struct{}
}

func NewSubnetListDB() *SubnetListDB {
	return &SubnetListDB{
		mu:   sync.RWMutex{},
		data: make(map[domain.ListType]map[string]struct{}),
	}
}

func (db *SubnetListDB) GetSubnetLists(_ context.Context, listType domain.ListType) ([]string, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	cidrsMap, exists := db.data[listType]
	if !exists {
		return []string{}, nil
	}

	cidrs := make([]string, 0, len(cidrsMap))
	for cidr := range cidrsMap {
		cidrs = append(cidrs, cidr)
	}
	return cidrs, nil
}

func (db *SubnetListDB) SaveSubnetList(_ context.Context, listType domain.ListType, cidrs []string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	cidrsMap := make(map[string]struct{})
	for _, cidr := range cidrs {
		cidrsMap[cidr] = struct{}{}
	}
	db.data[listType] = cidrsMap
	return nil
}

func (db *SubnetListDB) ClearSubnetList(_ context.Context, listType domain.ListType) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.data, listType)
	return nil
}

func (db *SubnetListDB) AddCIDRToSubnetList(_ context.Context, listType domain.ListType, cidr string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	cidrsMap, exists := db.data[listType]
	if !exists {
		cidrsMap = make(map[string]struct{})
		db.data[listType] = cidrsMap
	}
	cidrsMap[cidr] = struct{}{}
	return nil
}

func (db *SubnetListDB) RemoveCIDRFromSubnetList(_ context.Context, listType domain.ListType, cidr string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	cidrsMap, exists := db.data[listType]
	if !exists {
		return nil
	}
	delete(cidrsMap, cidr)
	return nil
}
