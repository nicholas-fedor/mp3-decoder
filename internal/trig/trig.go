// Copyright (c) Nicholas Fedor 2026 <nick@nickfedor.com>
// SPDX-License-Identifier: AGPL-3.0-or-later

package trig

import "math/big"

// Series constants.
const (
	// precision is the number of bits sums are computed to before rounding.
	precision = 320
	// turn is a full turn in units of π.
	turn = 2
	// seriesStep is how many factors each term of a series takes.
	seriesStep = 2
)

// Machin's formula, π = 16 atan(1/5) - 4 atan(1/239).
const (
	// machinFirst is the reciprocal of the first arctangent's argument.
	machinFirst = 5
	// machinFirstWeight is the first arctangent's weight.
	machinFirstWeight = 16
	// machinSecond is the reciprocal of the second arctangent's argument.
	machinSecond = 239
	// machinSecondWeight is the second arctangent's weight.
	machinSecondWeight = 4
)

// Cos returns cos(π n/d), rounded to the nearest float64.
//
// It reduces the angle to one turn and sums the Taylor series to 320 bits, with π from Machin's
// formula. A cosine that is 0 is exactly 0.
//
// Parameters:
//   - numerator: the angle's numerator n, in units of π.
//   - denominator: the angle's denominator d, above 0.
//
// Returns:
//   - float64: the cosine.
func Cos(numerator, denominator int) float64 {
	result, _ := cosine(numerator, denominator).Float64()

	return result
}

// Sec returns 1/cos(π n/d), the secant, rounded to the nearest float64.
//
// It divides 1 by the cosine Cos sums, at 320 bits, so the result is rounded only once.
//
// Parameters:
//   - numerator: the angle's numerator n, in units of π.
//   - denominator: the angle's denominator d, above 0.
//
// Returns:
//   - float64: the secant, infinite where the cosine is 0.
func Sec(numerator, denominator int) float64 {
	one := new(big.Float).SetPrec(precision).SetInt64(1)

	result, _ := one.Quo(one, cosine(numerator, denominator)).Float64()

	return result
}

// Sin returns sin(π n/d), rounded to the nearest float64, as the cosine of its complement.
//
// Parameters:
//   - numerator: the angle's numerator n, in units of π.
//   - denominator: the angle's denominator d, above 0.
//
// Returns:
//   - float64: the sine.
func Sin(numerator, denominator int) float64 {
	return Cos(denominator-turn*numerator, turn*denominator)
}

// cosine returns cos(π n/d) to 320 bits, from the Taylor series of the angle reduced to one turn.
//
// Parameters:
//   - numerator: the angle's numerator n, in units of π.
//   - denominator: the angle's denominator d, above 0.
//
// Returns:
//   - *[big.Float]: the cosine, exactly 0 where it is 0.
func cosine(numerator, denominator int) *big.Float {
	period := turn * denominator

	reduced := numerator % period
	if reduced < 0 {
		reduced += period
	}

	sum := new(big.Float).SetPrec(precision)

	if turn*reduced%denominator == 0 && turn*reduced/denominator%turn == 1 {
		return sum
	}

	x := new(big.Float).SetPrec(precision).Mul(pi(), big.NewFloat(float64(reduced)))
	x.Quo(x, big.NewFloat(float64(denominator)))

	square := new(big.Float).SetPrec(precision).Mul(x, x)
	term := new(big.Float).SetPrec(precision).SetInt64(1)

	sum.SetInt64(1)

	for factor := int64(1); term.Sign() != 0 && term.MantExp(nil) > -precision-8; factor += seriesStep {
		term.Mul(term, square)
		term.Quo(term, big.NewFloat(float64(factor*(factor+1))))
		term.Neg(term)
		sum.Add(sum, term)
	}

	return sum
}

// pi returns π from Machin's formula.
//
// Returns:
//   - *[big.Float]: π to 320 bits.
func pi() *big.Float {
	first := arctanInverse(machinFirst)
	first.Mul(first, big.NewFloat(machinFirstWeight))

	second := arctanInverse(machinSecond)
	second.Mul(second, big.NewFloat(machinSecondWeight))

	return first.Sub(first, second)
}

// arctanInverse returns atan(1/x) from its series, the sum of (-1)^k / ((2k+1) x^(2k+1)).
//
// Parameters:
//   - x: the reciprocal of the argument.
//
// Returns:
//   - *[big.Float]: the arctangent to 320 bits.
func arctanInverse(x int64) *big.Float {
	sum := new(big.Float).SetPrec(precision)
	power := new(big.Float).SetPrec(precision).Quo(big.NewFloat(1), big.NewFloat(float64(x)))
	square := big.NewFloat(float64(x * x))

	for odd, negative := int64(1), false; power.MantExp(nil) > -precision-8; odd, negative = odd+seriesStep, !negative {
		term := new(big.Float).SetPrec(precision).Quo(power, big.NewFloat(float64(odd)))
		if negative {
			sum.Sub(sum, term)
		} else {
			sum.Add(sum, term)
		}

		power.Quo(power, square)
	}

	return sum
}
