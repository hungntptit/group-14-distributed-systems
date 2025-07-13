package hash

import (
	"testing"
)

func TestHashRing_GetNodeForKey(t *testing.T) {
	peers := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}

	ring := NewHashRing(peers, 100)

	keys := []string{"apple", "banana", "carrot", "dog", "elephant"}

	for _, key := range keys {
		node := ring.GetNodeForKey(key)

		if node == "" {
			t.Errorf("Key %s returned empty node", key)
		}

		valid := false
		for _, peer := range peers {
			if node == peer {
				valid = true
				break
			}
		}
		if !valid {
			t.Errorf("Key %s mapped to unknown node %s", key, node)
		}
	}
}

func TestHashRing_Consistency(t *testing.T) {
	peers := []string{
		"http://localhost:8001",
		"http://localhost:8002",
		"http://localhost:8003",
	}
	ring := NewHashRing(peers, 100)

	key := "myKey"
	node1 := ring.GetNodeForKey(key)
	node2 := ring.GetNodeForKey(key)

	if node1 != node2 {
		t.Errorf("Inconsistent hash result: %s vs %s", node1, node2)
	}
}

func TestHashRing_NodeChange(t *testing.T) {
	peers := []string{
		"http://localhost:8001",
		"http://localhost:8002",
	}
	ring := NewHashRing(peers, 100)

	key := "criticalKey"
	initialNode := ring.GetNodeForKey(key)
	
	ring.AddNode("http://localhost:8003")

	newNode := ring.GetNodeForKey(key)

	if initialNode != newNode {
		t.Logf("Node changed after adding peer: %s → %s", initialNode, newNode)
	} else {
		t.Logf("Key stayed on same node after adding peer: %s", initialNode)
	}
}
