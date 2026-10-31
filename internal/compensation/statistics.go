package compensation

import (
	"math"
	"sort"
)

// Critical values of Student's t distribution for a two-sided 95% interval,
// indexed by degrees of freedom minus one.
var tCritical95 = []float64{
	12.706, 4.303, 3.182, 2.776, 2.571, 2.447, 2.365, 2.306, 2.262, 2.228,
	2.201, 2.179, 2.160, 2.145, 2.131, 2.120, 2.110, 2.101, 2.093, 2.086,
	2.080, 2.074, 2.069, 2.064, 2.060, 2.056, 2.052, 2.048, 2.045, 2.042,
}

// tCritical returns the 95% critical value. Beyond the table the distribution
// is close enough to normal that 1.96 is accurate to well under the precision
// of a caliper reading.
func tCritical(degreesOfFreedom int) float64 {
	if degreesOfFreedom > len(tCritical95) {
		return 1.96
	}
	return tCritical95[degreesOfFreedom-1]
}

// median returns the middle of numbers without reordering them.
func median(numbers []float64) float64 {
	sorted := append([]float64(nil), numbers...)
	sort.Float64s(sorted)
	middle := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[middle-1] + sorted[middle]) / 2
	}
	return sorted[middle]
}

// robustDeviation estimates the standard deviation from the median absolute
// value, so that the outliers being looked for do not inflate it. 1.4826 makes
// the estimate match the standard deviation for normally distributed values.
func robustDeviation(residuals []float64) float64 {
	magnitudes := make([]float64, 0, len(residuals))
	for _, residual := range residuals {
		magnitudes = append(magnitudes, math.Abs(residual))
	}
	return 1.4826 * median(magnitudes)
}
