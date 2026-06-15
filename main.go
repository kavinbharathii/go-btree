package main

import (
	"fmt"
	"slices"
	"strings"
)

const ORDER = 4 // each node has a max of 4 children
// so a max of 4 - 1 = 3 keys can be held by a node

type Node struct {
	keys     []int
	children []*Node
	isLeaf   bool
}

type BTree struct {
	root *Node
}

type SplitResult struct {
	splitHappened bool
	returnKey     int
	returnNode    *Node
}

type MergeResult struct {
	underflowOccurred bool
}

func NewNode(isLeaf bool) *Node {
	if isLeaf == true {
		return &Node{
			keys:     []int{},
			children: nil,
			isLeaf:   true,
		}
	}
	return &Node{
		keys:     []int{},
		children: []*Node{},
		isLeaf:   false,
	}
}

func overflowKeys(n *Node) bool {
	return len(n.keys) >= ORDER
}

func underflowKeys(n *Node) bool {
	return len(n.keys) < ((ORDER+1)/2)-1
}

func LastKey(n *Node) (int, error) {
	if len(n.keys) <= 0 {
		return 0, fmt.Errorf("no keys found for node")
	}
	return n.keys[len(n.keys)-1], nil
}

func LastChild(n *Node) (*Node, error) {
	if n.isLeaf == true {
		return nil, fmt.Errorf("node is not a leaf node")
	}
	if len(n.children) <= 0 {
		return nil, fmt.Errorf("no children found for node")
	}
	return n.children[len(n.children)-1], nil
}

func Search(node *Node, key int) bool {
	for ind, k := range node.keys {
		if k == key {
			return true
		} else if k > key {
			if node.isLeaf == false && len(node.children) > 0 {
				return Search(node.children[ind], key)
			}
			return false
		}
	}
	// if we reach here, it means key is greater than the max k of node
	if node.isLeaf {
		return false
	}
	lastChild, err := LastChild(node)
	if err != nil {
		return false
	}
	return Search(lastChild, key)
}

func SplitLeaf(n *Node) (*SplitResult, error) {
	if !n.isLeaf {
		return nil, fmt.Errorf("node is not a leaf node")
	}
	if !overflowKeys(n) {
		return &SplitResult{
			splitHappened: false,
			returnKey:     0,
			returnNode:    nil,
		}, nil
	}
	median := len(n.keys) / 2
	resKey := n.keys[median]
	resNode := NewNode(true)
	resNode.keys = append([]int{}, n.keys[median+1:]...)
	n.keys = n.keys[:median:median]
	return &SplitResult{
		splitHappened: true,
		returnKey:     resKey,
		returnNode:    resNode,
	}, nil
}

func SplitInternal(n *Node) (*SplitResult, error) {
	if n.isLeaf {
		return nil, fmt.Errorf("node is a leaf node")
	}
	if !overflowKeys(n) {
		return &SplitResult{
			splitHappened: false,
			returnKey:     0,
			returnNode:    nil,
		}, nil
	}
	median := len(n.keys) / 2
	resKey := n.keys[median]
	resNode := NewNode(false)
	resNode.keys = append([]int{}, n.keys[median+1:]...)
	n.keys = n.keys[:median:median]
	resNode.children = append([]*Node{}, n.children[median+1:]...)
	n.children = n.children[: median+1 : median+1]
	return &SplitResult{
		splitHappened: true,
		returnKey:     resKey,
		returnNode:    resNode,
	}, nil
}

func mergeLeaf(n *Node, separator int) (*MergeResult, error) {
	leftChild := n.children[separator]
	rightChild := n.children[separator+1]

	if !leftChild.isLeaf || !rightChild.isLeaf {
		return nil, fmt.Errorf("cannot perform mergeLeaf: children nodes are not leaves")
	}

	resChild := NewNode(true)

	resChild.keys = append(leftChild.keys, n.keys[separator])
	resChild.keys = append(resChild.keys, rightChild.keys...)

	// replace the left and right nodes in the parent's
	// children with the resChild
	n.keys = append(n.keys[:separator], n.keys[separator+1:]...)
	n.children = append(n.children[:separator], append([]*Node{resChild}, n.children[separator+2:]...)...)
	return &MergeResult{
		underflowOccurred: underflowKeys(n),
	}, nil
}

func mergeInternal(n *Node, separator int) (*MergeResult, error) {
	leftChild := n.children[separator]
	rightChild := n.children[separator+1]

	if leftChild.isLeaf || rightChild.isLeaf {
		return nil, fmt.Errorf("cannot perform mergeInternal: children nodes are leaves")
	}

	resChild := NewNode(false)
	resChild.keys = append([]int{}, leftChild.keys...)
	resChild.keys = append(resChild.keys, n.keys[separator])
	resChild.keys = append(resChild.keys, rightChild.keys...)
	resChild.children = append([]*Node{}, leftChild.children...)
	resChild.children = append(resChild.children, rightChild.children...)

	n.keys = append(n.keys[:separator], n.keys[separator+1:]...)
	n.children = append(n.children[:separator], append([]*Node{resChild}, n.children[separator+2:]...)...)

	return &MergeResult{
		underflowOccurred: underflowKeys(n),
	}, nil
}

func borrowFromLeft(n *Node, childIndex int) {
	child := n.children[childIndex]
	leftSibling := n.children[childIndex-1]

	// bring parent separator down into child
	child.keys = append([]int{n.keys[childIndex-1]}, child.keys...)
	// move left sibling's last key up to parent
	n.keys[childIndex-1] = leftSibling.keys[len(leftSibling.keys)-1]
	leftSibling.keys = leftSibling.keys[:len(leftSibling.keys)-1]

	// if internal, move left sibling's last child over
	if !child.isLeaf {
		child.children = append([]*Node{leftSibling.children[len(leftSibling.children)-1]}, child.children...)
		leftSibling.children = leftSibling.children[:len(leftSibling.children)-1]
	}
}

func borrowFromRight(n *Node, childIndex int) {
	child := n.children[childIndex]
	rightSibling := n.children[childIndex+1]

	// bring parent separator down into child
	child.keys = append(child.keys, n.keys[childIndex])
	// move right sibling's first key up to parent
	n.keys[childIndex] = rightSibling.keys[0]
	rightSibling.keys = rightSibling.keys[1:]

	// if internal, move right sibling's first child over
	if !child.isLeaf {
		child.children = append(child.children, rightSibling.children[0])
		rightSibling.children = rightSibling.children[1:]
	}
}

func fixUnderflow(n *Node, childIndex int) (*MergeResult, error) {
	child := n.children[childIndex]

	if !underflowKeys(child) {
		return &MergeResult{underflowOccurred: false}, nil
	}

	// try borrow from left sibling
	if childIndex > 0 && len(n.children[childIndex-1].keys) > ((ORDER+1)/2)-1 {
		borrowFromLeft(n, childIndex)
		return &MergeResult{underflowOccurred: false}, nil
	}

	// try borrow from right sibling
	if childIndex < len(n.children)-1 && len(n.children[childIndex+1].keys) > ((ORDER+1)/2)-1 {
		borrowFromRight(n, childIndex)
		return &MergeResult{underflowOccurred: false}, nil
	}

	// merge
	if childIndex > 0 {
		// merge with left sibling
		if child.isLeaf {
			return mergeLeaf(n, childIndex-1)
		}
		return mergeInternal(n, childIndex-1)
	}
	// merge with right sibling
	if child.isLeaf {
		return mergeLeaf(n, childIndex)
	}
	return mergeInternal(n, childIndex)
}

func insert(n *Node, key int) (*SplitResult, error) {
	if n.isLeaf {
		insertInd := len(n.keys)
		for ind, val := range n.keys {
			if val >= key {
				insertInd = ind
				break
			}
		}

		n.keys = slices.Insert(n.keys, insertInd, key) // insert given key at the correct index in leaf node
		splitRes, err := SplitLeaf(n)
		if err != nil {
			return nil, err
		}
		return splitRes, nil
	}

	// if the node is internal
	var targetChildIndex int
	foundChild := false
	for ind, val := range n.keys {
		if key < val {
			targetChildIndex = ind
			foundChild = true
			break
		}
	}

	if !foundChild {
		targetChildIndex = len(n.keys)
	}

	splitRes, err := insert(n.children[targetChildIndex], key)
	if err != nil {
		return nil, err
	}

	if !splitRes.splitHappened {
		return splitRes, nil
	}

	insertInd := len(n.keys)
	for ind, val := range n.keys {
		if val >= splitRes.returnKey {
			insertInd = ind
			break
		}
	}

	n.keys = slices.Insert(n.keys, insertInd, splitRes.returnKey)
	n.children = slices.Insert(n.children, insertInd+1, splitRes.returnNode)

	splitRes, err = SplitInternal(n)
	if err != nil {
		return nil, err
	}

	return splitRes, nil
}

func InsertIntoBtree(bt *BTree, key int) error {
	splitRes, err := insert(bt.root, key)
	if err != nil {
		return err
	}

	if !splitRes.splitHappened {
		return nil
	}

	newRoot := NewNode(false)
	newRoot.keys = []int{splitRes.returnKey}
	newRoot.children = []*Node{bt.root, splitRes.returnNode}

	bt.root = newRoot
	return nil
}

func deleteNode(n *Node, key int) (*MergeResult, error) {
	// find key in current node
	keyIndex := -1
	for i, k := range n.keys {
		if k == key {
			keyIndex = i
			break
		}
	}

	if n.isLeaf {
		if keyIndex == -1 {
			return nil, fmt.Errorf("key %d not found", key)
		}
		n.keys = append(n.keys[:keyIndex], n.keys[keyIndex+1:]...)
		return &MergeResult{underflowOccurred: underflowKeys(n)}, nil
	}

	// key found in internal node — replace with in-order successor
	if keyIndex != -1 {
		// find in-order successor (smallest key in right subtree)
		successor := n.children[keyIndex+1]
		for !successor.isLeaf {
			successor = successor.children[0]
		}
		successorKey := successor.keys[0]

		// replace key with successor
		n.keys[keyIndex] = successorKey

		// delete successor from right subtree
		mergeRes, err := deleteNode(n.children[keyIndex+1], successorKey)
		if err != nil {
			return nil, err
		}

		if !mergeRes.underflowOccurred {
			return &MergeResult{underflowOccurred: false}, nil
		}

		return fixUnderflow(n, keyIndex+1)
	}

	// key not in this node — find correct child to recurse into
	targetChildIndex := len(n.keys)
	for i, k := range n.keys {
		if key < k {
			targetChildIndex = i
			break
		}
	}

	mergeRes, err := deleteNode(n.children[targetChildIndex], key)
	if err != nil {
		return nil, err
	}

	if !mergeRes.underflowOccurred {
		return &MergeResult{underflowOccurred: false}, nil
	}

	return fixUnderflow(n, targetChildIndex)
}

func DeleteFromBtree(bt *BTree, key int) error {
	mergeRes, err := deleteNode(bt.root, key)
	if err != nil {
		return err
	}

	// if root is now empty, its only child becomes the new root
	if len(bt.root.keys) == 0 && !bt.root.isLeaf {
		bt.root = bt.root.children[0]
	}

	_ = mergeRes
	return nil
}

func PrintTree(n *Node, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Printf("%sLeaf=%v Keys=%v\n", indent, n.isLeaf, n.keys)
	for _, child := range n.children {
		PrintTree(child, depth+1)
	}
}

func main() {
	bt := &BTree{root: NewNode(true)}

	for i := 1; i <= 10; i++ {
		InsertIntoBtree(bt, i)
	}

	fmt.Println("--- initial tree ---")
	PrintTree(bt.root, 0)

	for _, key := range []int{5, 3, 7, 1, 9} {
		fmt.Printf("\n--- deleting %d ---\n", key)
		err := DeleteFromBtree(bt, key)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		PrintTree(bt.root, 0)
	}

	fmt.Println("\n--- search after deletes ---")
	for _, key := range []int{2, 4, 5, 6, 8, 10} {
		fmt.Printf("Search %d: %v\n", key, Search(bt.root, key))
	}
}
