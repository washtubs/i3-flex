package main

import "testing"

func TestGoldenRatioNormalize(t *testing.T) {
	res := goldenRatio.Normalize()
	if res != 619 {
		t.Fail()
	}
}
