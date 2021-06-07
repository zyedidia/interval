// Package interval provides an interval tree backed by an AVL tree. In addition,
// the interval tree supports a lazy shifting algorithm.
package interval

type key struct {
	id  int
	pos int
}

// compare orders keys by pos and then id.
func (k key) compare(other key) int {
	if k.pos < other.pos {
		return -1
	} else if k.pos > other.pos {
		return 1
	} else if k.id < other.id {
		return -1
	} else if k.id > other.id {
		return 1
	}
	return 0
}

type Tree struct {
	root *node
}

// Adds the given interval to the tree. An id can also be given to the interval
// to separate different types of intervals.
func (t *Tree) Add(id, low, high int, val Value) {
	t.root = t.root.add(key{id, low}, high, val)
}

// FindLargest returns the largest interval associated with (id, pos).
func (t *Tree) FindLargest(id, pos int) Value {
	n := t.root.search(key{id, pos})
	if n == nil || len(n.iv) == 0 {
		return nil
	}

	var max, maxi int
	for i := range n.iv {
		if n.iv[i].interval.High > max {
			max = n.iv[i].interval.High
			maxi = i
		}
	}
	return n.iv[maxi].value
}

// RemoveAndShift removes all entries that overlap with [low, high) and then shifts
// all entries greater than low by amt.
func (t *Tree) RemoveAndShift(low, high, amt int) {
	t.root = t.root.removeOverlaps(low, high)
	if amt != 0 {
		t.root.addShift(shift{low, amt})
	}
}

// Size returns the number of intervals in the tree.
func (t *Tree) Size() int {
	return t.root.getSize()
}

type ivalue struct {
	interval Interval
	value    Value
}

// A shift of intervals in the tree. The shift starts at idx and moves
// intervals after idx by amt. Shifts are lazily applied in the tree to avoid
// frequent linear time costs.
type shift struct {
	idx int
	amt int
}

type node struct {
	key    key
	max    int
	iv     []ivalue
	shifts []shift

	// height counts nodes (not edges)
	height int
	left   *node
	right  *node
}

func (n *node) addShift(sh shift) {
	if n == nil {
		return
	}

	n.shifts = append(n.shifts, sh)
}

func (n *node) applyShifts() {
	if n == nil {
		return
	}
	for _, sh := range n.shifts {
		if n.max >= sh.idx {
			if n.key.pos >= sh.idx {
				n.key.pos += sh.amt
				for i, iv := range n.iv {
					n.iv[i].interval = iv.interval.Shift(sh.amt)
				}
			}
			n.max += sh.amt
			// n.updateMax()
		}

		n.left.addShift(sh)
		n.right.addShift(sh)
	}
	n.shifts = nil
}

func (n *node) add(key key, high int, value Value) *node {
	if n == nil {
		return &node{
			key: key,
			max: high,
			iv: []ivalue{ivalue{
				interval: Interval{key.pos, high},
				value:    value,
			}},
			height: 1,
			left:   nil,
			right:  nil,
		}
	}
	n.applyShifts()

	if key.compare(n.key) < 0 {
		n.left = n.left.add(key, high, value)
	} else if key.compare(n.key) > 0 {
		n.right = n.right.add(key, high, value)
	} else {
		// if same key exists update value
		n.iv = append(n.iv, ivalue{
			interval: Interval{key.pos, high},
			value:    value,
		})
	}
	return n.rebalanceTree()
}

func (n *node) calcMax() int {
	max := 0
	for _, iv := range n.iv {
		if iv.interval.High > max {
			max = iv.interval.High
		}
	}
	return max
}

func (n *node) updateMax() {
	if n != nil {
		if n.right != nil {
			n.max = max(n.max, n.right.max)
		}
		if n.left != nil {
			n.max = max(n.max, n.left.max)
		}
		n.max = max(n.max, n.calcMax())
	}
}

func (n *node) remove(key key) *node {
	if n == nil {
		return nil
	}
	n.applyShifts()
	if key.compare(n.key) < 0 {
		n.left = n.left.remove(key)
	} else if key.compare(n.key) > 0 {
		n.right = n.right.remove(key)
	} else {
		if n.left != nil && n.right != nil {
			n.left.applyShifts()
			n.right.applyShifts()
			// node to delete found with both children;
			// replace values with smallest node of the right sub-tree
			rightMinNode := n.right.findSmallest()
			n.key = rightMinNode.key
			n.iv = rightMinNode.iv
			n.shifts = rightMinNode.shifts
			// delete smallest node that we replaced
			n.right = n.right.remove(rightMinNode.key)
		} else if n.left != nil {
			n.left.applyShifts()
			// node only has left child
			n = n.left
		} else if n.right != nil {
			n.right.applyShifts()
			// node only has right child
			n = n.right
		} else {
			// node has no children
			n = nil
			return n
		}

	}
	return n.rebalanceTree()
}

func (n *node) search(key key) *node {
	if n == nil {
		return nil
	}
	n.applyShifts()
	if key.compare(n.key) < 0 {
		return n.left.search(key)
	} else if key.compare(n.key) > 0 {
		return n.right.search(key)
	} else {
		return n
	}
}

func (n *node) overlaps(low, high int, result []Value) []Value {
	if n == nil {
		return result
	}

	n.applyShifts()

	if low >= n.max {
		return result
	}

	result = n.left.overlaps(low, high, result)

	for _, iv := range n.iv {
		if Overlaps(iv.interval, Interval{low, high}) {
			result = append(result, iv.value)
		}
	}

	if high <= n.key.pos {
		return result
	}

	result = n.right.overlaps(low, high, result)
	return result
}

func (n *node) removeOverlaps(low, high int) *node {
	if n == nil {
		return n
	}

	n.applyShifts()

	if low >= n.max {
		return n
	}

	n.left = n.left.removeOverlaps(low, high)

	for i := 0; i < len(n.iv); {
		if Overlaps(n.iv[i].interval, Interval{low, high}) {
			n.iv[i] = n.iv[len(n.iv)-1]
			n.iv = n.iv[:len(n.iv)-1]
		} else {
			i++
		}
	}

	if len(n.iv) == 0 {
		doright := high > n.key.pos
		n = n.remove(n.key)
		if doright {
			return n.removeOverlaps(low, high)
		}
		return n
	}

	if high <= n.key.pos {
		return n
	}

	n.right = n.right.removeOverlaps(low, high)
	return n
}

func (n *node) getHeight() int {
	if n == nil {
		return 0
	}
	return n.height
}

func (n *node) getSize() int {
	if n == nil {
		return 0
	}
	return n.left.getSize() + n.right.getSize() + 1
}

func (n *node) updateHeightAndMax() {
	n.height = 1 + max(n.left.getHeight(), n.right.getHeight())
	n.updateMax()
}

// Checks if node is balanced and rebalance
func (n *node) rebalanceTree() *node {
	if n == nil {
		return n
	}
	n.updateHeightAndMax()

	// check balance factor and rotateLeft if right-heavy and rotateRight if left-heavy
	balanceFactor := n.left.getHeight() - n.right.getHeight()
	if balanceFactor == -2 {
		// check if child is left-heavy and rotateRight first
		if n.right.left.getHeight() > n.right.right.getHeight() {
			n.right = n.right.rotateRight()
		}
		return n.rotateLeft()
	} else if balanceFactor == 2 {
		// check if child is right-heavy and rotateLeft first
		if n.left.right.getHeight() > n.left.left.getHeight() {
			n.left = n.left.rotateLeft()
		}
		return n.rotateRight()
	}
	return n
}

// Rotate nodes left to balance node
func (n *node) rotateLeft() *node {
	n.applyShifts()
	if n.right != nil {
		n.right.applyShifts()
	}

	newRoot := n.right
	n.right = newRoot.left
	newRoot.left = n

	n.updateHeightAndMax()
	newRoot.updateHeightAndMax()
	return newRoot
}

// Rotate nodes right to balance node
func (n *node) rotateRight() *node {
	n.applyShifts()
	if n.left != nil {
		n.left.applyShifts()
	}
	newRoot := n.left
	n.left = newRoot.right
	newRoot.right = n

	n.updateHeightAndMax()
	newRoot.updateHeightAndMax()
	return newRoot
}

// Finds the smallest child (based on the key) for the current node
func (n *node) findSmallest() *node {
	if n.left != nil {
		n.left.applyShifts()
		return n.left.findSmallest()
	} else {
		return n
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
