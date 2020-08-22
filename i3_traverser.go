package main

import (
	"fmt"
	"log"

	"go.i3wm.org/i3/v4"
)

// Resumable node traverser
type Traverser struct {
	path          []*i3.Node
	pathPositions []int
}

type TraverserHook func(path []*i3.Node, node *i3.Node)

// Does a depth first traversal, calling onPush *after* pushing, onPop *before* popping, and onLeaf for each leaf node
func (t *Traverser) depthFirstTraversal(onPush TraverserHook, onPop TraverserHook, onLeaf TraverserHook) {
	for {
		top := len(t.path) - 1
		node := t.path[top]
		start := t.pathPositions[top]
		if len(t.path) != len(t.pathPositions) {
			panic("path and pathPositions different lengths")
		}
		for i := start; i < len(node.Nodes); i++ {
			v := node.Nodes[i]
			t.pathPositions[top] = i + 1
			if len(v.Nodes) > 0 {
				t.path = append(t.path, v)
				t.pathPositions = append(t.pathPositions, 0)
				onPush(t.path, v)
				break
			}
			onLeaf(t.path, v)
		}

		if top+1 == len(t.path) { // We did not descend
			onPop(t.path, node)
			newLength := len(t.path) - 1
			t.path = t.path[:newLength]
			t.pathPositions = t.pathPositions[:newLength]
		}

		// Only the root element remains, and it's exhausted
		if len(t.path) == 1 && t.pathPositions[len(t.path)-1] >= len(t.path[len(t.path)-1].Nodes) {
			break
		}
	}
}

func createTraverser(node *i3.Node) *Traverser {
	path := make([]*i3.Node, 0)
	pathPositions := make([]int, 0) // represents the first unprocessed value for the path element
	path = append(path, node)
	pathPositions = append(pathPositions, 0) // start at the 0th node of the top corresponding path element
	return &Traverser{path, pathPositions}
}

func simplePrint(t *Traverser) {

	baseIndent := 2
	indent := baseIndent

	onPop := func(path []*i3.Node, node *i3.Node) {
		if isSplitContainer(node) {
			indent = indent - 2
		}
	}
	onPush := func(path []*i3.Node, node *i3.Node) {
		if isSplitContainer(node) {
			fmt.Printf("+%s> %s[%d]\n", dashes(indent), string(node.Layout), node.ID)
			indent = indent + 2
		}
	}
	onLeaf := func(path []*i3.Node, node *i3.Node) {
		hasSplitAncestor := indent > baseIndent
		if hasSplitAncestor && isWindow(node) { // aka we have a split parent
			fmt.Printf("+%s> %s[%d]\n", dashes(indent), "window", node.ID)
		}
	}

	t.depthFirstTraversal(onPush, onPop, onLeaf)

}

func fullUpdate(t *Traverser) []FlexUpdate {

	updates := make([]FlexUpdate, 0)

	onPop := func(path []*i3.Node, node *i3.Node) {}
	onLeaf := func(path []*i3.Node, node *i3.Node) {}
	onPush := func(path []*i3.Node, node *i3.Node) {
		if !isSplitContainer(node) {
			return
		}
		for _, n := range node.Nodes {
			if n.Type == i3.WorkspaceNode {
				// No workspace containers
				return
			}
		}
		var (
			sizer func(node *i3.Node) int
			dir   FlexDirection
		)
		if node.Layout == i3.SplitH {
			sizer = func(node *i3.Node) int { return int(node.Rect.Width) } // TODO checked conversion?
			dir = Horizontal
		} else { // SplitV
			sizer = func(node *i3.Node) int { return int(node.Rect.Height) } // TODO checked conversion?
			dir = Vertical
		}
		sum := sizer(node)
		//log.Printf("PATH %+v", path)
		//log.Printf("NODE %+v", node)
		//log.Printf("SUM dir=[%s] children=[%d] sum=[%d]", dir, len(node.Nodes), sum)
		items := make([]FlexItemUpdate, 0, len(node.Nodes))
		for _, n := range node.Nodes {
			size := sizer(n)
			//log.Printf("CHILD NODE %+v", n)
			//log.Printf("size %d", size)
			sum = sum - size
			items = append(items, FlexItemUpdate{
				ExternalId: n.ID,
				Size:       size,
			})
		}
		// TODO: This is a result of my configs and gaps I suspect.
		// I need to understand why the sum isn't 0
		expectedDifference := 114 + 22*(len(node.Nodes)-1)
		if sum != expectedDifference {
			log.Print(fmt.Sprintf("Sizes did not add up. difference=[%d]", sum))
		}
		updates = append(updates, FlexUpdate{
			ExternalId: node.ID,
			Items:      items,
			Direction:  dir,
		})
	}

	t.depthFirstTraversal(onPush, onPop, onLeaf)

	return updates
}
