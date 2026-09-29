package sharding

import (
	"fmt"
	"math"
	"strconv"
	"testing"
)

const keys = 200_000

func nodes(n int) []Node {
	result := make([]Node, n)

	for i := range result {
		result[i] = Node{ID: fmt.Sprintf("wallet-%d", i+1), Weight: 1}
	}

	return result
}

func TestMod_SpreadsKeysEvenly(t *testing.T) {
	strategy := NewMod(nodes(3))
	counts := map[string]int{}

	for id := 1; id <= keys; id++ {
		counts[strategy.Locate(strconv.Itoa(id)).ID]++
	}

	for node, count := range counts {
		share := float64(count) / keys
		if math.Abs(share-1.0/3) > 0.01 {
			t.Errorf("%s owns %.3f of keys, want about 0.333", node, share)
		}
	}
}

func TestMod_AddingNodeMovesMostKeys(t *testing.T) {
	before := NewMod(nodes(3))
	after := NewMod(nodes(4))
	moved := 0

	for id := 1; id <= keys; id++ {
		key := strconv.Itoa(id)
		if before.Locate(key).ID != after.Locate(key).ID {
			moved++
		}
	}

	share := float64(moved) / keys
	if math.Abs(share-0.75) > 0.01 {
		t.Fatalf("moved %.3f of keys, want about 0.75", share)
	}
}
