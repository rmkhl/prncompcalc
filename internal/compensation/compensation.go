package compensation

import (
	"errors"
	"fmt"
	"math"
)

const (
	// A residual beyond this many robust standard deviations is an outlier.
	// With the floor below this flags about 1% of clean six-point runs in
	// simulation, independent of the number of points, and finds every 0.3 mm
	// misreading.
	outlierDeviations = 3.5
	// Quantised caliper readings can make the robust deviation tiny or zero, so
	// nothing closer to the line than this is an outlier.
	outlierFloorMM = 0.05
	// Below this many used points the intervals are too wide to mean much.
	recommendedPoints = 4
	// Two sizes always fit a straight line, so three are needed to see one.
	recommendedSizes = 3
	// The largest acceptable slope uncertainty: 0.2 mm over a 100 mm part.
	maxSlopeUncertainty = 0.002
)

// Outlier is a measurement left out of the fit.
type Outlier struct {
	Line     int
	Expected float64
	Actual   float64
	Residual float64
}

// AxisFit is the fit of the measurements of a single axis.
type AxisFit struct {
	Feature Feature
	Line    Line
}

// Report is the compensation calculated from a set of measurements.
type Report struct {
	Line
	PointsUsed int
	Sizes      int
	MinSize    float64
	MaxSize    float64
	Outliers   []Outlier
	// ContourComp is the per-side change to outer contours in mm. Positive
	// makes a contour larger.
	ContourComp float64
	// HoleComp is the per-side change to holes in mm. Positive makes a hole
	// larger. It is meaningful only when HolesMeasured is above zero.
	HoleComp      float64
	HolesMeasured int
	Axes          []AxisFit
	Warnings      []string
}

// ScaleFactor is what to scale a model by to cancel the shrinkage.
func (r *Report) ScaleFactor() float64 {
	return 1 / r.Slope
}

func splitOuterAndHoles(measurements []*Measurement) (outer, holes []*Measurement) {
	for _, measurement := range measurements {
		if measurement.Feature == FeatureHole {
			holes = append(holes, measurement)
		} else {
			outer = append(outer, measurement)
		}
	}
	return outer, holes
}

func removeOutliers(measurements []*Measurement) (kept []*Measurement, outliers []Outlier) {
	slope, intercept := theilSen(measurements)
	residuals := make([]float64, 0, len(measurements))
	for _, m := range measurements {
		residuals = append(residuals, m.Actual-(slope*m.Expected+intercept))
	}
	limit := math.Max(outlierDeviations*robustDeviation(residuals), outlierFloorMM)
	for i, m := range measurements {
		if math.Abs(residuals[i]) > limit {
			outliers = append(outliers, Outlier{m.Line, m.Expected, m.Actual, residuals[i]})
		} else {
			kept = append(kept, m)
		}
	}
	return kept, outliers
}

// holeComp uses the slope of the outer fit because the scaling that cancels
// shrinkage applies to holes too. What remains after it is the hole's own
// offset, which outer dimensions say nothing about: polygon faceting and bead
// squeeze make holes print small whatever the outer contour does.
func holeComp(holes []*Measurement, slope float64) float64 {
	offsets := make([]float64, 0, len(holes))
	for _, hole := range holes {
		offsets = append(offsets, hole.Actual-slope*hole.Expected)
	}
	return -median(offsets) / 2
}

func fitAxes(measurements []*Measurement) []AxisFit {
	var axes []AxisFit
	for _, feature := range []Feature{FeatureX, FeatureY} {
		var onAxis []*Measurement
		for _, measurement := range measurements {
			if measurement.Feature == feature {
				onAxis = append(onAxis, measurement)
			}
		}
		if line, err := fitLine(onAxis); err == nil {
			axes = append(axes, AxisFit{feature, line})
		}
	}
	return axes
}

func (r *Report) describeSizes(measurements []*Measurement) {
	r.PointsUsed = len(measurements)
	r.Sizes = distinctSizes(measurements)
	r.MinSize, r.MaxSize = math.Inf(1), math.Inf(-1)
	for _, measurement := range measurements {
		r.MinSize = math.Min(r.MinSize, measurement.Expected)
		r.MaxSize = math.Max(r.MaxSize, measurement.Expected)
	}
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Report) addQualityWarnings() {
	if r.PointsUsed < recommendedPoints {
		r.warn("only %d points used, measure at least %d", r.PointsUsed, recommendedPoints)
	}
	if r.Sizes < recommendedSizes {
		r.warn("only %d sizes measured, %d are needed to show the error is linear", r.Sizes, recommendedSizes)
	}
	if uncertainty := r.SlopeRange.HalfWidth(); uncertainty > maxSlopeUncertainty {
		r.warn("shrinkage is uncertain by ±%.2f mm per 100 mm, measure a wider range of sizes or more points", uncertainty*100)
	}
	if r.InterceptRange.Contains(0) {
		r.warn("the offset cannot be told apart from zero, so the contour compensation may be noise")
	}
}

// Calculate fits the outer measurements, leaving out outliers, and derives the
// compensation from them. The outlier screen and the final fit are separate
// steps so a clean data set gets the plain least squares result, with
// confidence intervals that actually describe it.
func Calculate(measurements []*Measurement) (*Report, error) {
	outer, holes := splitOuterAndHoles(measurements)
	if len(outer) == 0 {
		return nil, errors.New("no outer measurements to fit")
	}
	if distinctSizes(outer) < 2 {
		return nil, errors.New("at least 2 different sizes are needed, they are what separates shrinkage from offset")
	}

	kept, outliers := removeOutliers(outer)
	line, err := fitLine(kept)
	if err != nil {
		return nil, err
	}

	report := &Report{
		Line:          line,
		Outliers:      outliers,
		ContourComp:   -line.Intercept / 2,
		HolesMeasured: len(holes),
		Axes:          fitAxes(kept),
	}
	if len(holes) > 0 {
		report.HoleComp = holeComp(holes, line.Slope)
	}
	report.describeSizes(kept)
	report.addQualityWarnings()
	return report, nil
}
