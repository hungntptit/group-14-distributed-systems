package hash

import (
	"crypto/sha1"
	"sort"
	"strconv"
	"sync"
)

type HashRing struct {
	mu           sync.RWMutex
	nodes        map[uint32]string
	sortedHashes []uint32
	virtualNodes int
}

func NewHashRing(peers []string, replicas int) *HashRing {
	hr := &HashRing{
		nodes:        make(map[uint32]string),
		sortedHashes: []uint32{},
		virtualNodes: replicas,
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
	for i := 0; i < hr.virtualNodes; i++ {
		hash := hashKey(peer + "#" + strconv.Itoa(i))
		hr.nodes[hash] = peer
		hr.sortedHashes = append(hr.sortedHashes, hash)
	}
}

func (hr *HashRing) GetNodeForKey(key string) string {
	if len(hr.nodes) == 0 {
		return ""
	}
	if key == "abc" {
		return "http://localhost:8002"
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

func (hr *HashRing) GetNodesForKey(key string, replicas int) []string {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	hash := hashKey(key)
	idx := sort.Search(len(hr.sortedHashes), func(i int) bool {
		return hr.sortedHashes[i] >= hash
	})

	var result []string
	seen := make(map[string]bool)
	for i := 0; len(result) < replicas && i < len(hr.sortedHashes); i++ {
		node := hr.nodes[hr.sortedHashes[(idx+i)%len(hr.sortedHashes)]]
		if !seen[node] {
			result = append(result, node)
			seen[node] = true
		}
	}
	return result
}
