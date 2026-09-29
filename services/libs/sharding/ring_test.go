package sharding

import (
	"math"
	"strconv"
	"testing"
)

func shares(strategy Strategy) map[string]float64 {
	counts := map[string]int{}

	for id := 1; id <= keys; id++ {
		counts[strategy.Locate(strconv.Itoa(id)).ID]++
	}

	result := map[string]float64{}

	for node, count := range counts {
		result[node] = float64(count) / keys
	}

	return result
}

func TestRing_AddingNodeMovesKeysOnlyToIt(t *testing.T) {
	before := NewRing(nodes(3), 1)
	after := NewRing(nodes(4), 1)
	moved := 0

	for id := 1; id <= keys; id++ {
		key := strconv.Itoa(id)
		from, to := before.Locate(key).ID, after.Locate(key).ID

		if from == to {
			continue
		}

		if to != "wallet-4" {
			t.Fatalf("key %s moved from %s to %s, not to the new node", key, from, to)
		}

		moved++
	}

	movedShare := float64(moved) / keys
	newNodeShare := shares(after)["wallet-4"]

	if math.Abs(movedShare-newNodeShare) > 1e-9 {
		t.Fatalf("moved %.4f of keys, the new node owns %.4f", movedShare, newNodeShare)
	}
}

func TestRing_MoreVirtualNodesEvenOutShares(t *testing.T) {
	spread := func(vnodes int) float64 {
		low, high := 1.0, 0.0

		for _, share := range shares(NewRing(nodes(4), vnodes)) {
			low = min(low, share)
			high = max(high, share)
		}

		return high - low
	}

	if coarse, fine := spread(1), spread(256); fine >= coarse || fine > 0.05 {
		t.Fatalf("spread with 1 point %.3f, with 256 points %.3f", coarse, fine)
	}
}

func TestRing_WeightDoublesShare(t *testing.T) {
	weighted := nodes(4)
	weighted[3].Weight = 2

	got := shares(NewRing(weighted, 256))
	ratio := got["wallet-4"] / got["wallet-1"]

	if ratio < 1.7 || ratio > 2.3 {
		t.Fatalf("node with weight 2 owns %.2f times more than a node with weight 1", ratio)
	}
}

func TestRing_PointWrapsAround(t *testing.T) {
	ring := NewRing(nodes(3), 2)

	if ring.Size() != 6 {
		t.Fatalf("size = %d, want 6", ring.Size())
	}

	last := ring.points[len(ring.points)-1]

	for id := 1; id <= keys; id++ {
		key := strconv.Itoa(id)

		if Hash(key) > last.Hash {
			if point := ring.Point(key); point != ring.points[0] {
				t.Fatalf("key after the last point must go to the first one, got %+v", point)
			}

			return
		}
	}

	t.Fatal("no key hashed past the last point")
}
