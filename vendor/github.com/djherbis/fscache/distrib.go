package fscache

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
)

// Distributor provides a way to partition keys into Caches.
type Distributor interface {

	// GetCache will always return the same Cache for the same key.
	GetCache(key string) Cache

	// Clean should wipe all the caches this Distributor manages
	Clean() error
}

// stdDistribution distributes the keyspace evenly.
func stdDistribution(key string, n uint64) uint64 {
	h := sha1.New()
	io.WriteString(h, key)
	buf := bytes.NewBuffer(h.Sum(nil)[:8])
	i, _ := binary.ReadUvarint(buf)
	return i % n
}

// NewDistributor returns a Distributor which evenly distributes the keyspace
// into the passed caches.
func NewDistributor(caches ...Cache) Distributor {
	if len(caches) == 0 {
		return nil
	}
	return &distrib{
		distribution: stdDistribution,
		caches:       caches,
		size:         uint64(len(caches)),
	}
}

type distrib struct {
	distribution func(key string, n uint64) uint64
	caches       []Cache
	size         uint64
}

func (d *distrib) GetCache(key string) Cache {
	return d.caches[d.distribution(key, d.size)]
}

// BUG(djherbis): Return an error if cleaning fails
func (d *distrib) Clean() error {
	for _, c := range d.caches {
		c.Clean()
	}
	return nil
}

// NewPartition returns a Cache which uses the Caches defined by the passed Distributor.
func NewPartition(d Distributor) Cache {
	return &partition{
		distributor: d,
	}
}

type partition struct {
	distributor Distributor
}

func (p *partition) Get(key string) (ReadAtCloser, io.WriteCloser, error) {
	return p.distributor.GetCache(key).Get(key)
}

func (p *partition) Remove(key string) error {
	return p.distributor.GetCache(key).Remove(key)
}

func (p *partition) Exists(key string) bool {
	return p.distributor.GetCache(key).Exists(key)
}

func (p *partition) Clean() error {
	return p.distributor.Clean()
}
