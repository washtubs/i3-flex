package main

import "go.i3wm.org/i3/v4"

type FlexUpdate struct {
	ExternalId i3.NodeID
	Direction  FlexDirection
	Items      []FlexItemUpdate
}

type FlexItemUpdate struct {
	ExternalId i3.NodeID
	Size       int
}
