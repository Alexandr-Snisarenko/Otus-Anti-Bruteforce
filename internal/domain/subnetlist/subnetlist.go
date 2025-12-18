package subnetlist

import (
	"context"
	"errors"
	"net"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ports"
)

// SubnetList - реализация ведения списка подсетей в CIDR нотации.
// Для каждого типа списка (listType) отдельный экземпляр SubnetList.
// Пример типов списков: whitelist, blacklist.
// Пример использования:
//		whitelist := subnetlist.NewSubnetList(ports.Whitelist)
//		blacklist := subnetlist.NewSubnetList(ports.Blacklist)
//
// Сети хранятся в памяти в виде map[string]*net.IPNet для быстрого поиска.
// Ключ - CIDR нотация сети в виде строки (например, "192.168.1.0/24").
// Значение - указатель на net.IPNet, представляющий эту же сеть.
// Проверка принадлежности IP к подсетям осуществляется прямым перебором всех подсетей.
// Предполагается, что количество подсетей в списке невелико (до нескольких сотен), поэтому
// производительность такого подхода будет приемлемой.
// Может использоваться совместно с SubnetRepo для загрузки/сохранения списков подсетей.
// Для этого есть метод Load(ctx, repo).
// Или может использоваться автономно, без постоянного хранения.

// SubnetList представляет список подсетей в CIDR нотации.
type SubnetList struct {
	listType domain.ListType
	nets     map[string]*net.IPNet
}

func NewSubnetList(listType domain.ListType) *SubnetList {
	return &SubnetList{
		listType: listType,
		nets:     make(map[string]*net.IPNet),
	}
}

func (nl *SubnetList) Add(cidr string) error {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.Join(ErrInvalidCIDR, err)
	}

	// Проверка корректности CIDR. IP адрес должен быть адресом подсети.
	if !ip.Equal(ipnet.IP) {
		return ErrInvalidCIDR
	}

	nl.nets[cidr] = ipnet
	return nil
}

func (nl *SubnetList) Contains(ip net.IP) bool {
	for _, net := range nl.nets {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

func (nl *SubnetList) Remove(cidr string) {
	delete(nl.nets, cidr)
}

func (nl *SubnetList) Clear() {
	nl.nets = make(map[string]*net.IPNet)
}

func (nl *SubnetList) Load(ctx context.Context, repo ports.SubnetRepo) error {
	cidrs, err := repo.GetSubnetLists(ctx, nl.listType)
	if err != nil {
		return err
	}

	nl.Clear()
	for _, cidr := range cidrs {
		if err := nl.Add(cidr); err != nil {
			return err
		}
	}

	return nil
}
