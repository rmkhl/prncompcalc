package compensation

import "math"

// Interval is a 95% confidence interval.
type Interval struct {
	Low  float64
	High float64
}

// Contains reports whether value lies inside the interval.
func (i Interval) Contains(value float64) bool {
	return i.Low <= value && value <= i.High
}

// HalfWidth returns the distance from the middle of the interval to its ends.
func (i Interval) HalfWidth() float64 {
	return (i.High - i.Low) / 2
}

// Line is the fit actual = Slope*expected + Intercept.
type Line struct {
	Slope           float64
	Intercept       float64
	SlopeRange      Interval
	InterceptRange  Interval
	ResidualScatter float64
}

func distinctSizes(measurements []*Measurement) int {
	sizes := make(map[float64]struct{}, len(measurements))
	for _, measurement := range measurements {
		sizes[measurement.Expected] = struct{}{}
	}
	return len(sizes)
}

// fitLine is an ordinary least squares fit. A proportional error and a constant
// one have to be separated by fitting them together: a ratio of actual to
// expected is s + c/expected, which counts any offset as extra shrinkage and
// more so on small parts.
func fitLine(measurements []*Measurement) (Line, error) {
	count := len(measurements)
	if count < 3 {
		return Line{}, newMessage(CodeTooFewToFit, 3)
	}
	if distinctSizes(measurements) < 2 {
		return Line{}, newMessage(CodeOneSize)
	}

	var sumExpected, sumActual float64
	for _, m := range measurements {
		sumExpected += m.Expected
		sumActual += m.Actual
	}
	meanExpected := sumExpected / float64(count)
	meanActual := sumActual / float64(count)

	var spread, covariance float64
	for _, m := range measurements {
		spread += (m.Expected - meanExpected) * (m.Expected - meanExpected)
		covariance += (m.Expected - meanExpected) * (m.Actual - meanActual)
	}
	slope := covariance / spread
	intercept := meanActual - slope*meanExpected

	var squaredResiduals float64
	for _, m := range measurements {
		residual := m.Actual - (slope*m.Expected + intercept)
		squaredResiduals += residual * residual
	}
	degreesOfFreedom := count - 2
	scatter := math.Sqrt(squaredResiduals / float64(degreesOfFreedom))
	critical := tCritical(degreesOfFreedom)
	slopeMargin := critical * scatter / math.Sqrt(spread)
	interceptMargin := critical * scatter * math.Sqrt(1/float64(count)+meanExpected*meanExpected/spread)

	return Line{
		Slope:           slope,
		Intercept:       intercept,
		SlopeRange:      Interval{slope - slopeMargin, slope + slopeMargin},
		InterceptRange:  Interval{intercept - interceptMargin, intercept + interceptMargin},
		ResidualScatter: scatter,
	}, nil
}

// theilSen fits the median slope over every pair of points. Unlike least
// squares it is barely moved by a mistyped value, which is what makes its
// residuals usable for finding one. The caller guarantees two distinct sizes.
func theilSen(measurements []*Measurement) (slope, intercept float64) {
	var slopes []float64
	for i, a := range measurements {
		for _, b := range measurements[i+1:] {
			if a.Expected != b.Expected {
				slopes = append(slopes, (b.Actual-a.Actual)/(b.Expected-a.Expected))
			}
		}
	}
	slope = median(slopes)

	offsets := make([]float64, 0, len(measurements))
	for _, m := range measurements {
		offsets = append(offsets, m.Actual-slope*m.Expected)
	}
	return slope, median(offsets)
}
