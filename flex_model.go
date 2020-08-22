package main

import (
	"log"

	"go.i3wm.org/i3/v4"
)

func isFlexed(size Size) bool {
	return int(size) > (normal/2)+1
}

type MinConstraint interface {
	Invalidate()
}

type MinItemConstraint interface {
	MinConstraint
	ItemIndex() int
}

type MinConstraintChain struct {
	userDefined []MinItemConstraint
	global      []MinConstraint
}

// Invalidates the first available constraint
func (chain *MinConstraintChain) Invalidate() bool {
	if len(chain.userDefined) > 0 {
		log.Printf("Invalidating user defined minimum")
		chain.userDefined[0].Invalidate()
		chain.userDefined = chain.userDefined[1:]
		return true
	} else if len(chain.global) > 0 {
		log.Printf("Invalidating global minimum")
		chain.global[0].Invalidate()
		chain.global = chain.global[1:]
		return true
	} else {
		log.Printf("Chain exhausted. Hard limits observed.")
		return false
	}
}

type FlexItemMinFlexConstraint struct {
	*FlexItem
	idx int
}

func (c FlexItemMinFlexConstraint) ItemIndex() int { return c.idx }
func (c FlexItemMinFlexConstraint) Invalidate()    { c.FlexItem.softMinFlex = -1 }

type FlexItemMinUnflexConstraint struct {
	*FlexItem
	idx int
}

func (c FlexItemMinUnflexConstraint) ItemIndex() int { return c.idx }
func (c FlexItemMinUnflexConstraint) Invalidate()    { c.FlexItem.softMinUnflex = -1 }

type GlobalSoftMinUnflexConstraint struct{ *FlexModel }

func (c GlobalSoftMinUnflexConstraint) Invalidate() { c.FlexModel.globalSoftMinUnflexObserved = false }

type GlobalSoftMinFlexConstraint struct{ *FlexModel }

func (c GlobalSoftMinFlexConstraint) Invalidate() { c.FlexModel.globalSoftMinFlexObserved = false }

// The direction in which the items in a model are resized
type FlexDirection string

const (
	Horizontal FlexDirection = "horizontal" // corresponds to splith
	Vertical   FlexDirection = "vertical"   // corresponds to splitv
)

type FlexModel struct {
	globals     GlobalSizings
	id          i3.NodeID
	items       []*FlexItem
	constraints []MinItemConstraint // User defined constraints
	direction   FlexDirection

	globalSoftMinUnflexObserved bool
	globalSoftMinFlexObserved   bool
}

type FlexEvent struct {
	id       i3.NodeID
	increase Size
}

// Called in response to user initiated resizing.
// The user may have increased the size of an item without flexing it
// Or the user may have manually flexed an item.
// If the user also focused a new window, that will be handled later in OnFocus.
func (f *FlexModel) OnUpdate(events []FlexEvent) {
	// First update those items, corresponding to those IDs,
	// and track the amount that will need to be reduced from other items

	delta := Size(0)
	excludeIndexes := make([]int, 0, len(events))
	for _, ev := range events {
		for k, item := range f.items {
			if item.id == ev.id {
				item.current = item.current + ev.increase
				delta = delta - ev.increase
				excludeIndexes = append(excludeIndexes, k)
				f.putConstraint(k) // create or bump a min constraint based on whether it's currently flexed
				break
			}
		}
	}

	// Do the reduction loop
	f.reductionLoop(delta, excludeIndexes, 0)
}

// Performs the reduction loop, invalidating constraints as needed to make the
// necessary reductions incurred by the grown items (denoted by excludeIndexes).
//
// FlexItem.current should already be updated for excludeIndexes before running the loop
func (f *FlexModel) reductionLoop(delta Size, excludeIndexes []int, callStackPosition int) {

	if callStackPosition > 1 {
		panic("reductionLoop shouldn't recurse more than once")
	}

	if delta >= 0 {
		panic("Delta must be negative")
	}

	// Try to observe global min constraints again
	f.globalSoftMinFlexObserved = true
	f.globalSoftMinUnflexObserved = true

	// Get relevant constraints,
	// i.e. all constraints not pertaining to the items which expanded
	// Notably globalSoftMinFlex is irrelevant when we are expanding a flexed item

	userDefinedConstraints := make([]MinItemConstraint, 0)
	for _, c := range f.constraints {
		found := false
		for _, v := range excludeIndexes {
			if c.ItemIndex() == v {
				found = true
				break
			}
		}
		if !found { // not one of the excludeIndexes items
			userDefinedConstraints = append(userDefinedConstraints, c)
		}
	}

	globalConstraints := make([]MinConstraint, 0)
	globalConstraints = append(globalConstraints, GlobalSoftMinUnflexConstraint{f})
	flexing := false
	for _, v := range excludeIndexes {
		if isFlexed(f.items[v].current) {
			flexing = true
		}
	}
	if flexing {
		globalConstraints = append(globalConstraints, GlobalSoftMinFlexConstraint{f})
	}

	chain := MinConstraintChain{
		userDefined: userDefinedConstraints,
		global:      globalConstraints,
	}

	// Do the actual loop with invalidations
	compl := f.complement(excludeIndexes)
	rem := delta
	proceed := true
	for proceed {
		rem = rebalance(f.sizes(compl), f.minimums(compl), rem)
		proceed = rem != 0 && chain.Invalidate()
	}

	f.constraints = chain.userDefined

	// Reset global state: note this strangeness is ultimately due to FlexModel implementing GetMin itself
	// As opposed to the chain for example, which may be more appropriate
	f.globalSoftMinFlexObserved = true
	f.globalSoftMinUnflexObserved = true

	// If after cycling through all constraints and rebalancing,
	// there is still some remaining (i.e. hard limits prevent further growth)
	// adjust the flexed items back down
	if rem < 0 {
		log.Printf("Invalidating constraints wasn't enough. These are too large. Reducing back.")
		f.reductionLoop(rem, compl, callStackPosition+1)
	}

}

func (f *FlexModel) Flex(idx int) bool {
	toFlex := f.items[idx]
	if isFlexed(toFlex.current) {
		// Already flexed. Nothing to do
		return false
	}

	for _, v := range f.items {
		log.Printf("preflex item current %d", v.current)
	}
	delta := Size(0)
	excludeIndexes := make([]int, 0, 2)
	excludeIndexes = append(excludeIndexes, idx)

	// Add to the size of the element that will be flexed
	// Subtract from the delta
	var newFlexSize Size
	if toFlex.softMinFlex > 0 {
		newFlexSize = toFlex.softMinFlex
	} else {
		newFlexSize = f.globals.softMinFlex
	}
	delta = delta - (newFlexSize - toFlex.current)
	toFlex.current = newFlexSize
	log.Printf("New flex size %d", toFlex.current)

	// Subtract from the size of the element that will be unflexed
	// Add to the delta
	for k, item := range f.items {
		if k == idx {
			continue
		}
		// Shrink this item down
		// Should just be one
		if isFlexed(item.current) {
			min := f.globals.softMinUnflex
			if item.softMinUnflex > 0 {
				min = item.softMinUnflex
			}
			delta = delta + (item.current - min)
			item.current = min
			excludeIndexes = append(excludeIndexes, k)
		}
	}
	for _, v := range f.items {
		log.Printf("With flex current %d", v.current)
	}
	if len(excludeIndexes) > 2 { // sanity
		log.Print("Warning: More than one item was shrunken, indicating multiple flexed items.")
	}

	if delta > 0 {
		// Shrunk more: distribute among unflexed
		unflexed := f.unflexed()
		rebalance(f.sizes(unflexed), f.minimums(unflexed), delta)
	} else if delta < 0 {
		// Grew more: proceed with reduction loop
		log.Printf("delta! %d", delta)
		f.reductionLoop(delta, excludeIndexes, 0)
	}
	log.Printf("delta %d", delta)
	for _, v := range f.items {
		log.Printf("postflex item current %d", v.current)
	}
	return true
}

func (f *FlexModel) sizes(indexes []int) []*Size {
	sizes := make([]*Size, 0, len(indexes))
	for _, idx := range indexes {
		sizes = append(sizes, &f.items[idx].current)
	}
	return sizes
}

func (f *FlexModel) minimums(indexes []int) []Size {
	mins := make([]Size, 0, len(indexes))
	for _, idx := range indexes {
		mins = append(mins, f.GetMin(idx))
	}
	return mins
}

func (f *FlexModel) unflexed() []int {
	unflexed := make([]int, 0, len(f.items)-1)
	for k, v := range f.items {
		if !isFlexed(v.current) {
			unflexed = append(unflexed, k)
		}
	}
	return unflexed
}

// Adds or bumps the constraint for the item.
//
// If the item is flexed, adds a FlexItemMinFlexConstraint
// If the item is unflexed, adds a FlexItemMinUnflexConstraint
// If such a constraint exists, it is simply bumped to the top
func (f *FlexModel) putConstraint(idx int) {
	item := f.items[idx]
	isFlexConstraint := isFlexed(item.current)
	bumpIdx := -1
	for k, constraint := range f.constraints {
		if idx == constraint.ItemIndex() {
			_, flexConstraintExists := constraint.(FlexItemMinFlexConstraint)
			_, unflexConstraintExists := constraint.(FlexItemMinUnflexConstraint)
			if flexConstraintExists && isFlexConstraint {
				// bump flex
				bumpIdx = k
			} else if unflexConstraintExists && !isFlexConstraint {
				// bump unflex
				bumpIdx = k
			}
		}
	}
	if bumpIdx > -1 {
		bumpConstraint := f.constraints[bumpIdx]
		f.constraints = append(f.constraints[0:bumpIdx], f.constraints[bumpIdx:]...)
		f.constraints = append(f.constraints, bumpConstraint)
	} else {
		if isFlexConstraint {
			f.constraints = append(f.constraints, FlexItemMinFlexConstraint{item, idx})
		} else {
			f.constraints = append(f.constraints, FlexItemMinUnflexConstraint{item, idx})
		}
	}
}

// Gets the complement of the given indexes.
// O(n^2) but n is realistically very small (<5)
func (f *FlexModel) complement(indexes []int) []int {
	compl := make([]int, 0, len(f.items)-len(indexes))
	for i := 0; i < len(f.items); i++ {
		found := false
		for _, v := range indexes {
			if v == i {
				found = true
				break
			}
		}
		if !found {
			compl = append(compl, i)
		}
	}
	return compl
}

func (f *FlexModel) GetMin(idx int) Size {
	item := f.items[idx]
	if isFlexed(item.current) {
		if item.softMinFlex > 0 {
			return item.softMinFlex
		} else if f.globalSoftMinFlexObserved {
			return f.globals.softMinFlex
		} else {
			return f.globals.hardMinFlex
		}
	} else {
		if item.softMinUnflex > 0 {
			return item.softMinUnflex
		} else if f.globalSoftMinUnflexObserved {
			return f.globals.softMinUnflex
		} else {
			return f.globals.hardMinUnflex
		}
	}
}

type FlexItem struct {
	id      i3.NodeID
	current Size

	// Stores user overrides for flex items
	// <= 0 means unspecified
	softMinFlex   Size
	softMinUnflex Size
}
