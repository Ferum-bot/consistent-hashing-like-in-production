package sharding

import (
	"cmp"
	"slices"
	"strconv"
)

type RingPoint struct {
	Hash  uint64
	Node  Node
	Index int
}

type Ring struct {
	points []RingPoint
}

func NewRing(nodes []Node, vnodes int) *Ring {
	vnodes = max(vnodes, 1)
	points := make([]RingPoint, 0, len(nodes)*vnodes)

	for _, node := range nodes {
		count := vnodes * max(node.Weight, 1)

		for i := range count {
			points = append(points, RingPoint{Hash: Hash(PointKey(node.ID, i)), Node: node, Index: i})
		}
	}

	slices.SortFunc(points, func(a, b RingPoint) int {
		return cmp.Compare(a.Hash, b.Hash)
	})

	return &Ring{points: points}
}

func (r *Ring) Locate(key string) Node {
	return r.Point(key).Node
}

func (r *Ring) Point(key string) RingPoint {
	hash := Hash(key)

	i, _ := slices.BinarySearchFunc(r.points, hash, func(point RingPoint, target uint64) int {
		return cmp.Compare(point.Hash, target)
	})

	if i == len(r.points) {
		i = 0
	}

	return r.points[i]
}

func (r *Ring) Size() int {
	return len(r.points)
}

func PointKey(nodeID string, index int) string {
	return nodeID + "#" + strconv.Itoa(index)
}
