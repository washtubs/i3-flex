package main

const normal = 1000

// Fractional approximation for the golden ratio, useful for flex values.
// https://www.theproblemsite.com/ask/2017/09/approximation-for-the-golden-ratio
var goldenRatio = Ratio{
	1134903170, // fib(44)
	1836311903, // fib(45)
}

type Ratio struct {
	Dividend, Divisor int
}

// Returns the new dividend as if the Divisor was normal (1000)
func (r Ratio) Normalize() int {
	scaled := r.Dividend
	complement := r.Divisor - r.Dividend
	rescale([]*int{&scaled, &complement}, r.Divisor, normal)
	return scaled
}
