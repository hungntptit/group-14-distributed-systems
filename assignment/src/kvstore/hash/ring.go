package hash

import (
	"crypto/sha1"
	"sort"
	"strconv"
)

type HashRing struct {
	nodes        map[uint32]string
	sortedHashes []uint32
	replicas     int
}

func NewHashRing(peers []string, replicas int) *HashRing {
	hr := &HashRing{
		nodes:        make(map[uint32]string),
		sortedHashes: []uint32{},
		replicas:     replicas,
	}

	for _, peer := range peers {
		hr.AddNode(peer)
	}

	sort.Slice(hr.sortedHashes, func(i, j int) bool {
		return hr.sortedHashes[i] < hr.sortedHashes[j]
	})

	return hr
}

func hashKey(key string) uint32 {
	h := sha1.Sum([]byte(key))
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}

func (hr *HashRing) AddNode(peer string) {
	for i := 0; i < hr.replicas; i++ {
		hash := hashKey(peer + "#" + strconv.Itoa(i))
		hr.nodes[hash] = peer
		hr.sortedHashes = append(hr.sortedHashes, hash)
	}
}

func (hr *HashRing) GetNodeForKey(key string) string {
	if len(hr.nodes) == 0 {
		return ""
	}
	hash := hashKey(key)
	idx := sort.Search(len(hr.sortedHashes), func(i int) bool {
		return hr.sortedHashes[i] >= hash
	})
	if idx == len(hr.sortedHashes) {
		idx = 0
	}
	return hr.nodes[hr.sortedHashes[idx]]
}
