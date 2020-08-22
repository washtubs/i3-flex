package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"go.i3wm.org/i3/v4"
)

func isWindow(n *i3.Node) bool {
	return n.Window != 0
}

func isSplitContainer(n *i3.Node) bool {
	return !isWindow(n) && n.Layout == i3.SplitH || n.Layout == i3.SplitV
}

func dashes(n int) string {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		buf[i] = '-'
	}
	return string(buf)
}

func doPrint(tree i3.Tree) {
	tree.Root.FindChild(func(node *i3.Node) bool {
		//log.Printf("Visiting node win=%d nodeId=%d layout=%s type=%s len=%d", node.Window, node.ID, node.Layout, node.Type, len(node.Nodes))
		if isSplitContainer(node) {
			fmt.Printf("+--> %s %d\n", node.Layout, node.ID)
		} else if isWindow(node) {
			fmt.Printf("+-----> %d\n", node.Window)
		}
		return false
	})

}

func testRender() {
}

func serve() {
	fm := initFlexModels()
	fm.RegisterRenderer(&i3FlexRenderer{})

	rcv := i3.Subscribe(i3.WindowEventType)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for rcv.Next() {
			event := rcv.Event()
			ev, ok := event.(*i3.WindowEvent)
			if !ok {
				log.Printf("Unexpected event type: %+v, %+v", ev, event)
				continue
			}
			if ev.Change == "focus" {
				log.Printf("Got focus")
			}
			// Do updates
			tree, err := i3.GetTree()
			if err != nil {
				panic(err.Error())
			}
			t := createTraverser(tree.Root)
			updates := fullUpdate(t)
			fm.Updates(updates, true)
			fm.OnFocus(ev.Container.ID)
		}
		err := rcv.Close()
		if err != nil {
			log.Printf("Error closing i3 receiver: %s", err.Error())
		}
		wg.Done()
	}()

	wg.Wait()
}

func main() {
	globals := flag.NewFlagSet("", flag.ExitOnError)
	globals.Parse(os.Args[1:])
	commandStr := globals.Arg(0)

	if commandStr == "" {
		log.Fatal("No command") // TODO: list commands
	}
	//cmdArgs := globals.Args()[1:]
	switch commandStr {
	case "serve":
		serve()
	case "debug":
		testRender()
		tree, err := i3.GetTree()
		if err != nil {
			panic(err.Error())
		}
		t := createTraverser(tree.Root)
		simplePrint(t)
	}

	// Simple:
	//
	// Init all FlexModels
	// Wait for events
	// On event:
	//   On focus, render() the flex model to (both) current container(s)
	//

	// Stateful: Try to remember when a user grows a window manually
	//
	// Init all FlexModels
	// Wait for events
	// On event:
	//   Check FlexModel invalidate to revert to default sizing for example if the user added a window
	//   Check for user initiated resize
	//     If a user manually flexed an element (made it the largest in the container)
	//       Set a new flexed and complement for that element
	//     If a user manually increased the size of an element without flexing it
	//       Set the unflexed for this element and rebalance unflexed for all others scaled to the complement
	//   On focus, render() the flex model to (both) current container(s)

	// Traverse the tree for SplitH and SplitV layouts
	// Their immediate children can be treated as flex arrangements
	// I think we can subsribe to window events and update the arrangements purely from that
	// The window event will always tell us what the new container is, so we can use that to track new containers as well
	// If we detect a container no longer has windows we should prune it ourselves

}
