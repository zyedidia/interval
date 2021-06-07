package interval

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func randrange(max int) (int, int) {
	low := rand.Intn(max)
	high := low + rand.Intn(1000)
	if low == high {
		high = low + 1
	}
	return low, high
}

func check(a, b []Value, t *testing.T) {
	ai := make([]int, len(a))
	bi := make([]int, len(b))
	for i := range a {
		ai[i] = a[i].(int)
	}
	for i := range b {
		bi[i] = b[i].(int)
	}

	sort.Slice(ai, func(i, j int) bool {
		return ai[i] < ai[j]
	})
	sort.Slice(bi, func(i, j int) bool {
		return bi[i] < bi[j]
	})
	if len(a) != len(b) {
		t.Errorf("different len: %d vs %d", len(a), len(b))
		fmt.Println(ai)
		fmt.Println(bi)
		return
	}
	for i := range ai {
		if ai[i] != bi[i] {
			t.Errorf("error:\n%v\n%v", ai, bi)
		}
	}
}

func randint(min, max int) int {
	return rand.Intn(max-min) + min
}

func TestTree(t *testing.T) {
	it := &Tree{}
	ia := &Array{}

	const (
		opAdd = iota
		opFind
		opRemoveAndShift

		nops     = 50000
		maxidx   = 100000
		maxid    = 10
		maxshamt = 50
	)

	for i := 0; i < nops; i++ {
		op := rand.Intn(3)
		switch op {
		case opAdd:
			id := rand.Intn(maxid)
			low, high := randrange(maxidx)
			it.Add(id, low, high, i)
			ia.Add(id, low, high, i)
		case opFind:
			id := rand.Intn(maxid)
			pos := rand.Intn(maxidx)

			vt := it.FindLargest(id, pos)
			va := ia.FindLargest(id, pos)

			if vt == nil && va == nil {
				continue
			}

			if vt == nil && va != nil || va == nil && vt != nil {
				t.Fatalf("Find (%d, %d): %v != %v", id, pos, vt, va)
			}

			if vt.(int) != va.(int) {
				t.Fatalf("Find (%d, %d): %d != %d", id, pos, vt.(int), va.(int))
			}
		case opRemoveAndShift:
			low, high := randrange(maxidx)
			amt := randint(-maxshamt, maxshamt)

			it.RemoveAndShift(low, high, amt)
			ia.RemoveAndShift(low, high, amt)
		}
	}
}
