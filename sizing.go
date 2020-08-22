package main

type Size int

type GlobalSizings struct {
	defaultFlexRatio Ratio
	maxFlex          Size
	softMinFlex      Size
	hardMinFlex      Size
	softMinUnflex    Size
	hardMinUnflex    Size
}

var globals GlobalSizings = GlobalSizings{}

func init() {
	globals.defaultFlexRatio = goldenRatio
	globals.maxFlex = Size(Ratio{9, 10}.Normalize())
	globals.softMinFlex = Size(goldenRatio.Normalize())
	globals.hardMinFlex = Size(Ratio{1, 2}.Normalize() + 1)
	globals.softMinUnflex = Size(Ratio{1, 10}.Normalize())
	globals.hardMinUnflex = Size(Ratio{1, 20}.Normalize())
}

type Sizing interface {
	GetSize(idx, outOf int, isFlexed bool) Size
}

// A sizing implementation which flexes the flexed element to a fixed flex size,
// and makes the remaining elements share from the remaining unflexed size
type SimpleSizing struct {
	flexed   Size
	unflexed Size
}

func (s *SimpleSizing) GetSize(idx, outOf int, isFlexed bool) Size {
	if isFlexed {
		return s.flexed
	}
	return s.unflexed
}

var defaultSizing = SimpleSizing{1000, 500}
