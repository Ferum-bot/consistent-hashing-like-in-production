package sharding

import "github.com/cespare/xxhash/v2"

type Strategy interface {
	Locate(key string) Node
}

func Hash(key string) uint64 {
	return xxhash.Sum64String(key)
}
