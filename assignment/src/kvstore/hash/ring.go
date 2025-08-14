package hash

import (
	"crypto/sha1"
	"kvstore/logging"
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
	return hr
}

func hashKey(key string) uint32 {
	h := sha1.Sum([]byte(key))
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}

func (hr *HashRing) AddNode(peer string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	for i := 0; i < hr.virtualNodes; i++ {
		hash := hashKey(peer + "#" + strconv.Itoa(i))
		hr.nodes[hash] = peer
		hr.sortedHashes = append(hr.sortedHashes, hash)
	}
	sort.Slice(hr.sortedHashes, func(i, j int) bool {
		return hr.sortedHashes[i] < hr.sortedHashes[j]
	})
	logging.Infof("Added node %v to hash ring %v", peer, hr.nodes)
}

func (hr *HashRing) RemoveNode(peer string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	for i := 0; i < hr.virtualNodes; i++ {
		hash := hashKey(peer + "#" + strconv.Itoa(i))
		delete(hr.nodes, hash)
	}
	hr.sortedHashes = make([]uint32, 0, len(hr.nodes))
	for hash := range hr.nodes {
		hr.sortedHashes = append(hr.sortedHashes, hash)
	}
	sort.Slice(hr.sortedHashes, func(i, j int) bool {
		return hr.sortedHashes[i] < hr.sortedHashes[j]
	})
	logging.Infof("Removed node %v from hash ring %v", peer, hr.nodes)
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

func (hr *HashRing) ContainsPeer(peer string) bool {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	for i := 0; i < hr.virtualNodes; i++ {
		hash := hashKey(peer + "#" + strconv.Itoa(i))
		if _, ok := hr.nodes[hash]; ok {
			return true
		}
	}
	return false
}

func (hr *HashRing) GetAllPeers() []string {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	peersSet := make(map[string]struct{})
	for _, peer := range hr.nodes {
		peersSet[peer] = struct{}{}
	}

	peers := make([]string, 0, len(peersSet))
	for peer := range peersSet {
		peers = append(peers, peer)
	}

	sort.Strings(peers)
	return peers
}
