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

	if !slices.IsSorted(n.keys) {
		t.Fatalf("keys not sorted: %v", n.keys)
	}

	if len(n.keys) > ORDER-1 {
		t.Fatalf("node overflow: keys=%v has more than %d keys", n.keys, ORDER-1)
	}
	minKeys := ((ORDER + 1) / 2) - 1
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

func buildTree(t *testing.T, keys []int) *BTree {
	t.Helper()
	bt := &BTree{root: NewNode(true)}
	for _, k := range keys {
		if err := InsertIntoBtree(bt, k); err != nil {
			t.Fatalf("insert %d failed: %v", k, err)
		}
	}
	return bt
}

// ---------- Insert tests ----------

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
	n.keys = []int{1, 2, 3, 4}
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
		t.Fatalf("in-order mismatch:\ngot:  %v\nwant: %v", keys, expected)
	}
}

func TestInsert_RandomOrder(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 5; trial++ {
		bt := &BTree{root: NewNode(true)}
		n := 50
		perm := rng.Perm(n)
		for _, v := range perm {
			key := v + 1
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
			t.Fatalf("trial %d: in-order mismatch:\ngot:  %v\nwant: %v", trial, keys, expected)
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

// ---------- Delete tests ----------

func TestDelete_LeafOnly_NoUnderflow(t *testing.T) {
	// tree small enough that no underflow occurs
	bt := buildTree(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	for _, key := range []int{2, 5, 8} {
		if err := DeleteFromBtree(bt, key); err != nil {
			t.Fatalf("delete %d failed: %v", key, err)
		}
		validateTree(t, bt)
		if Search(bt.root, key) {
			t.Fatalf("key %d still found after delete", key)
		}
	}
}

func TestDelete_NotFound(t *testing.T) {
	bt := buildTree(t, []int{1, 2, 3, 4, 5})
	err := DeleteFromBtree(bt, 99)
	if err == nil {
		t.Fatalf("expected error when deleting non-existent key")
	}
}

func TestDelete_AllKeys_Sequential(t *testing.T) {
	n := 20
	keys := make([]int, n)
	for i := range keys {
		keys[i] = i + 1
	}
	bt := buildTree(t, keys)

	// delete in ascending order
	for _, key := range keys {
		if err := DeleteFromBtree(bt, key); err != nil {
			t.Fatalf("delete %d failed: %v", key, err)
		}
		validateTree(t, bt)
		if Search(bt.root, key) {
			t.Fatalf("key %d still found after delete", key)
		}
	}

	var remaining []int
	collectKeys(bt.root, &remaining)
	if len(remaining) != 0 {
		t.Fatalf("expected empty tree, got keys: %v", remaining)
	}
}

func TestDelete_AllKeys_Descending(t *testing.T) {
	n := 20
	keys := make([]int, n)
	for i := range keys {
		keys[i] = i + 1
	}
	bt := buildTree(t, keys)

	for i := n; i >= 1; i-- {
		if err := DeleteFromBtree(bt, i); err != nil {
			t.Fatalf("delete %d failed: %v", i, err)
		}
		validateTree(t, bt)
		if Search(bt.root, i) {
			t.Fatalf("key %d still found after delete", i)
		}
	}

	var remaining []int
	collectKeys(bt.root, &remaining)
	if len(remaining) != 0 {
		t.Fatalf("expected empty tree, got keys: %v", remaining)
	}
}

func TestDelete_RandomOrder(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	for trial := 0; trial < 5; trial++ {
		n := 50
		keys := make([]int, n)
		for i := range keys {
			keys[i] = i + 1
		}
		bt := buildTree(t, keys)

		// shuffle delete order
		perm := rng.Perm(n)
		remaining := make([]int, n)
		copy(remaining, keys)

		for _, idx := range perm {
			key := idx + 1
			if err := DeleteFromBtree(bt, key); err != nil {
				t.Fatalf("trial %d: delete %d failed: %v", trial, key, err)
			}
			validateTree(t, bt)
			if Search(bt.root, key) {
				t.Fatalf("trial %d: key %d still found after delete", trial, key)
			}
		}

		var got []int
		collectKeys(bt.root, &got)
		if len(got) != 0 {
			t.Fatalf("trial %d: expected empty tree after all deletes, got: %v", trial, got)
		}
	}
}

func TestDelete_InternalNode_Key(t *testing.T) {
	bt := buildTree(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	allKeys := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	deleted := map[int]bool{}

	for _, key := range []int{3, 6, 9} {
		if err := DeleteFromBtree(bt, key); err != nil {
			t.Fatalf("delete internal key %d failed: %v", key, err)
		}
		validateTree(t, bt)
		deleted[key] = true

		for _, k := range allKeys {
			if deleted[k] {
				continue
			}
			if !Search(bt.root, k) {
				t.Fatalf("key %d missing after deleting %d", k, key)
			}
		}
	}
}

func TestDelete_TreeShrinks(t *testing.T) {
	// force tree to shrink in height by deleting enough keys to trigger root collapse
	bt := buildTree(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	initialDepth := 0
	cur := bt.root
	for !cur.isLeaf {
		initialDepth++
		cur = cur.children[0]
	}

	// delete most keys to force merges up to root
	for _, key := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		if err := DeleteFromBtree(bt, key); err != nil {
			t.Fatalf("delete %d failed: %v", key, err)
		}
		validateTree(t, bt)
	}

	finalDepth := 0
	cur = bt.root
	for !cur.isLeaf {
		finalDepth++
		cur = cur.children[0]
	}

	if finalDepth >= initialDepth {
		t.Fatalf("expected tree to shrink in height, initial=%d final=%d", initialDepth, finalDepth)
	}
}

func TestInsertDelete_Interleaved(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	bt := &BTree{root: NewNode(true)}
	inserted := map[int]bool{}

	for i := 0; i < 100; i++ {
		key := rng.Intn(50) + 1
		if inserted[key] {
			if err := DeleteFromBtree(bt, key); err != nil {
				t.Fatalf("delete %d failed: %v", key, err)
			}
			delete(inserted, key)
		} else {
			if err := InsertIntoBtree(bt, key); err != nil {
				t.Fatalf("insert %d failed: %v", key, err)
			}
			inserted[key] = true
		}
		validateTree(t, bt)
	}

	// verify final state
	for key := range inserted {
		if !Search(bt.root, key) {
			t.Fatalf("key %d missing from tree", key)
		}
	}
}
