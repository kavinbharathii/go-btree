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

func fullKeys(n *Node) bool {
	return len(n.keys) == ORDER-1
}

func overflowKeys(n *Node) bool {
	return len(n.keys) >= ORDER
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
		err := InsertIntoBtree(bt, i)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		fmt.Printf("--- after inserting %d ---\n", i)
		PrintTree(bt.root, 0)
	}

	// also sanity check Search
	fmt.Println("Search 7:", Search(bt.root, 7))
	fmt.Println("Search 11:", Search(bt.root, 11))
}
