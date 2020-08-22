package main

import (
	"fmt"
	"log"
)

// Tries to distribute the delta to all sizes, moving it towards 0.
// Modifies nums in place, respecting their corresponding mins
// Returns the remaining delta value
// In general it tries to just distribute the surplus of nums over mins evenly
func rebalance(nums []*Size, mins []Size, delta Size) Size {

	if delta == 0 {
		return 0
	}

	if delta > 0 {
		// Proportional growth
		numsCpy := copySizes(nums)
		oldScale := 0
		for _, num := range nums {
			oldScale = oldScale + int(*num)
		}
		rescale(numsCpy, oldScale, int(delta))
		for i, n := range numsCpy {
			*nums[i] = *nums[i] + Size(*n)
		}

		return 0
	} else {
		surpluses := make([]*int, len(nums))
		totalSurplus := 0
		log.Printf("rebalancing delta=%d", delta)
		for i, _ := range nums {
			log.Printf("rebalancing num=[%d] min=[%d]", *nums[i], mins[i])
		}

		for i, num := range nums {
			surpluses[i] = new(int)
			*surpluses[i] = int(*num - mins[i])
			totalSurplus = totalSurplus + *surpluses[i]
		}
		oldScale := totalSurplus
		newScale := totalSurplus + int(delta)
		remDelta := Size(0)
		if newScale < 0 {
			remDelta = Size(newScale)
			newScale = 0
		}
		rescale(surpluses, oldScale, newScale)
		for i, n := range surpluses {
			*nums[i] = mins[i] + Size(*n)
		}
		return remDelta
	}
}

func copySizes(nums []*Size) []*int {
	// Assert size
	ints := make([]*int, len(nums))
	for k, v := range nums {
		ints[k] = new(int)
		*ints[k] = (int)(*v)
	}
	return ints
}

func checkScaleSizes(nums []*Size) {
	// Assert size
	ints := make([]*int, len(nums))
	for k, v := range nums {
		ints[k] = (*int)(v)
	}
	checkScale(ints, normal)
}

func checkScale(nums []*int, oldScale int) {
	sum := 0
	for _, v := range nums {
		sum = sum + *v
	}
	if sum != oldScale {
		panic(fmt.Sprintf("Expected nums to sum to %d, got %d", oldScale, sum))
	}
}

func rescaleSizes(nums []*Size, newScale int) {
	// Assert size
	ints := make([]*int, len(nums))
	for k, v := range nums {
		ints[k] = (*int)(v)
	}
	rescale(ints, normal, newScale)
}

func rescale(nums []*int, oldScale, newScale int) {

	if oldScale == 0 {
		// if oldScale is 0, we infer it from the nums assuming they add up to their scale
		for _, v := range nums {
			oldScale = oldScale + *v
		}
	} else {
		total := 0
		for _, v := range nums {
			total = total + *v
		}
		if total != oldScale {
			panic("oldScale specifies but it doesnt add up")
		}
	}
	log.Printf("Scaling oldScale=[%d] newScale=[%d]", oldScale, newScale)
	for _, v := range nums {
		log.Printf("num=[%d]", *v)
	}
	for _, v := range nums {
		scaled := float64(*v) * (float64(newScale) / float64(oldScale))
		*v = int(scaled)
	}
	sum := 0
	for _, v := range nums {
		sum = sum + *v
	}
	// Since we floored it's possible we'll have some remaining, distribute evenly
	rem := newScale - sum
	if rem > 10 {
		log.Printf("hmm, David probably sucks at math")
		log.Printf("rem=%d", rem)
	}
	for rem > 0 {
		for _, v := range nums {
			*v = *v + 1
			rem--
			if rem == 0 {
				return
			}
		}
	}
	log.Printf("Done scaling")
	for _, v := range nums {
		log.Printf("num=[%d]", *v)
	}
}
