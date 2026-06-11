package main

import (
	"math/rand"
	"slices"
	"testing"
)

// ---------- Helper: validate B-tree invariants ----------

func validate(t *testing.T, n *Node, isRoot bool, depth int, leafDepth *int) {
	t.Helper()

	if n.isLeaf {
		if *leafDepth == -1 {
			*leafDepth = depth
		} else if *leafDepth != depth {
			t.Fatalf("leaf depth mismatch: expected %d, got %d (keys=%v)", *leafDepth, depth, n.keys)
		}
		if n.children != nil && len(n.children) != 0 {
			t.Fatalf("leaf node has children: %v", n.children)
		}
	} else {
		if len(n.children) != len(n.keys)+1 {
			t.Fatalf("invariant broken: len(children)=%d != len(keys)+1=%d (keys=%v)",
				len(n.children), len(n.keys)+1, n.keys)
		}
	}

	// keys must be sorted
	if !slices.IsSorted(n.keys) {
		t.Fatalf("keys not sorted: %v", n.keys)
	}

	// key count bounds (skip min-bound check for root)
	if len(n.keys) > ORDER-1 {
		t.Fatalf("node overflow: keys=%v has more than %d keys", n.keys, ORDER-1)
	}
	minKeys := (ORDER+1)/2 - 1
	if !isRoot && len(n.keys) < minKeys {
		t.Fatalf("node underflow: keys=%v has fewer than %d keys", n.keys, minKeys)
	}

	for _, child := range n.children {
		validate(t, child, false, depth+1, leafDepth)
	}
}

func validateTree(t *testing.T, bt *BTree) {
	t.Helper()
	leafDepth := -1
	validate(t, bt.root, true, 0, &leafDepth)
}

// collect all keys via in-order traversal
func collectKeys(n *Node, out *[]int) {
	if n.isLeaf {
		*out = append(*out, n.keys...)
		return
	}
	for i, k := range n.keys {
		collectKeys(n.children[i], out)
		*out = append(*out, k)
	}
	collectKeys(n.children[len(n.children)-1], out)
}

// ---------- Tests ----------

func TestSplitLeaf_NoOverflow(t *testing.T) {
	n := NewNode(true)
	n.keys = []int{1, 2}

	res, err := SplitLeaf(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.splitHappened {
		t.Fatalf("expected no split, got split with key=%d", res.returnKey)
	}
}

func TestSplitLeaf_Overflow(t *testing.T) {
	n := NewNode(true)
	n.keys = []int{1, 2, 3, 4} // ORDER=4, overflow at 4

	res, err := SplitLeaf(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.splitHappened {
		t.Fatalf("expected split to happen")
	}
	if res.returnKey != 3 {
		t.Fatalf("expected median key 3, got %d", res.returnKey)
	}
	if !slices.Equal(n.keys, []int{1, 2}) {
		t.Fatalf("expected left keys [1 2], got %v", n.keys)
	}
	if !slices.Equal(res.returnNode.keys, []int{4}) {
		t.Fatalf("expected right keys [4], got %v", res.returnNode.keys)
	}
	if !res.returnNode.isLeaf {
		t.Fatalf("expected returned node to be a leaf")
	}
}

func TestSplitLeaf_OnInternalNode_Errors(t *testing.T) {
	n := NewNode(false)
	n.keys = []int{1, 2, 3, 4}
	_, err := SplitLeaf(n)
	if err == nil {
		t.Fatalf("expected error when splitting internal node as leaf")
	}
}

func TestSplitInternal_Overflow(t *testing.T) {
	n := NewNode(false)
	n.keys = []int{1, 2, 3, 4}
	// 5 children needed for 4 keys
	for i := 0; i < 5; i++ {
		c := NewNode(true)
		c.keys = []int{i * 10}
		n.children = append(n.children, c)
	}

	res, err := SplitInternal(n)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.splitHappened {
		t.Fatalf("expected split to happen")
	}
	if res.returnKey != 3 {
		t.Fatalf("expected median key 3, got %d", res.returnKey)
	}
	if !slices.Equal(n.keys, []int{1, 2}) {
		t.Fatalf("expected left keys [1 2], got %v", n.keys)
	}
	if !slices.Equal(res.returnNode.keys, []int{4}) {
		t.Fatalf("expected right keys [4], got %v", res.returnNode.keys)
	}
	if len(n.children) != 3 {
		t.Fatalf("expected left node to have 3 children, got %d", len(n.children))
	}
	if len(res.returnNode.children) != 2 {
		t.Fatalf("expected right node to have 2 children, got %d", len(res.returnNode.children))
	}
	if res.returnNode.isLeaf {
		t.Fatalf("expected returned node to be internal")
	}
}

func TestSplitInternal_OnLeaf_Errors(t *testing.T) {
	n := NewNode(true)
	n.keys = []int{1, 2, 3, 4}
	_, err := SplitInternal(n)
	if err == nil {
		t.Fatalf("expected error when splitting leaf node as internal")
	}
}

func TestInsert_SequentialAscending(t *testing.T) {
	bt := &BTree{root: NewNode(true)}

	for i := 1; i <= 20; i++ {
		if err := InsertIntoBtree(bt, i); err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
		validateTree(t, bt)
	}

	var keys []int
	collectKeys(bt.root, &keys)
	expected := make([]int, 20)
	for i := range expected {
		expected[i] = i + 1
	}
	if !slices.Equal(keys, expected) {
		t.Fatalf("in-order traversal mismatch:\ngot:  %v\nwant: %v", keys, expected)
	}

	for i := 1; i <= 20; i++ {
		if !Search(bt.root, i) {
			t.Fatalf("expected to find key %d", i)
		}
	}
	if Search(bt.root, 21) {
		t.Fatalf("did not expect to find key 21")
	}
	if Search(bt.root, 0) {
		t.Fatalf("did not expect to find key 0")
	}
}

func TestInsert_SequentialDescending(t *testing.T) {
	bt := &BTree{root: NewNode(true)}

	for i := 20; i >= 1; i-- {
		if err := InsertIntoBtree(bt, i); err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
		validateTree(t, bt)
	}

	var keys []int
	collectKeys(bt.root, &keys)
	expected := make([]int, 20)
	for i := range expected {
		expected[i] = i + 1
	}
	if !slices.Equal(keys, expected) {
		t.Fatalf("in-order traversal mismatch:\ngot:  %v\nwant: %v", keys, expected)
	}

	for i := 1; i <= 20; i++ {
		if !Search(bt.root, i) {
			t.Fatalf("expected to find key %d", i)
		}
	}
}

func TestInsert_RandomOrder(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for trial := 0; trial < 5; trial++ {
		bt := &BTree{root: NewNode(true)}

		n := 50
		perm := rng.Perm(n)
		for _, v := range perm {
			key := v + 1 // 1..n
			if err := InsertIntoBtree(bt, key); err != nil {
				t.Fatalf("trial %d: insert %d failed: %v", trial, key, err)
			}
			validateTree(t, bt)
		}

		var keys []int
		collectKeys(bt.root, &keys)
		expected := make([]int, n)
		for i := range expected {
			expected[i] = i + 1
		}
		if !slices.Equal(keys, expected) {
			t.Fatalf("trial %d: in-order traversal mismatch:\ngot:  %v\nwant: %v", trial, keys, expected)
		}

		for i := 1; i <= n; i++ {
			if !Search(bt.root, i) {
				t.Fatalf("trial %d: expected to find key %d", trial, i)
			}
		}
		if Search(bt.root, n+1) {
			t.Fatalf("trial %d: did not expect to find key %d", trial, n+1)
		}
	}
}

func TestInsert_DuplicateKeys(t *testing.T) {
	bt := &BTree{root: NewNode(true)}

	for i := 1; i <= 10; i++ {
		if err := InsertIntoBtree(bt, i); err != nil {
			t.Fatalf("insert %d failed: %v", i, err)
		}
	}
	// insert duplicates of existing keys
	for i := 1; i <= 10; i++ {
		if err := InsertIntoBtree(bt, i); err != nil {
			t.Fatalf("duplicate insert %d failed: %v", i, err)
		}
		validateTree(t, bt)
	}

	var keys []int
	collectKeys(bt.root, &keys)
	if len(keys) != 20 {
		t.Fatalf("expected 20 keys (with duplicates), got %d: %v", len(keys), keys)
	}
}
