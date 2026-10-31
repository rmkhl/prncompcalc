package compensation

import (
	"bytes"
	"errors"
	"math"
	"strings"
	"testing"
)

// The spreadsheet this tool was compared with fits the same six points to these
// values, so they double as an independent check of the fit.
func sampleMeasurements() []*Measurement {
	return []*Measurement{
		{Line: 1, Expected: 20, Actual: 19.8},
		{Line: 2, Expected: 20, Actual: 19.83},
		{Line: 3, Expected: 30, Actual: 29.74},
		{Line: 4, Expected: 30, Actual: 29.76},
		{Line: 5, Expected: 40, Actual: 39.69},
		{Line: 6, Expected: 40, Actual: 39.71},
	}
}

func exactMeasurements(slope, intercept float64, sizes ...float64) []*Measurement {
	measurements := make([]*Measurement, 0, len(sizes))
	for i, size := range sizes {
		measurements = append(measurements, &Measurement{Line: i + 1, Expected: size, Actual: slope*size + intercept})
	}
	return measurements
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func hasWarning(report *Report, text string) bool {
	for _, warning := range report.Warnings {
		if strings.Contains(warning, text) {
			return true
		}
	}
	return false
}

func mustCalculate(t *testing.T, measurements []*Measurement) *Report {
	t.Helper()
	report, err := Calculate(measurements)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestReadMeasurements(t *testing.T) {
	input := "# a comment\n\n20 19.8\n30\t29,7 X\n  40   39.6  hole  \n"
	var warnings bytes.Buffer
	got, err := ReadMeasurements(strings.NewReader(input), &warnings)
	if err != nil {
		t.Fatal(err)
	}
	if warnings.Len() != 0 {
		t.Errorf("unexpected warnings: %s", warnings.String())
	}
	if len(got) != 3 {
		t.Fatalf("read %d measurements, want 3", len(got))
	}
	if got[0].Line != 3 || got[0].Feature != FeatureOuter {
		t.Errorf("first = %+v", got[0])
	}
	assertClose(t, "decimal comma", got[1].Actual, 29.7)
	if got[1].Feature != FeatureX || got[2].Feature != FeatureHole {
		t.Errorf("features = %q, %q", got[1].Feature, got[2].Feature)
	}
}

func TestReadMeasurementsSkipsInvalidLines(t *testing.T) {
	for _, line := range []string{"20", "1 2 x 4", "x 1", "1 x", "0 1", "1 0", "-1 1", "NaN 1", "Inf 1", "1 2 z"} {
		t.Run(line, func(t *testing.T) {
			var warnings bytes.Buffer
			got, err := ReadMeasurements(strings.NewReader("20 19.8\n"+line+"\n"), &warnings)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 {
				t.Errorf("read %d measurements, want 1", len(got))
			}
			if !strings.HasPrefix(warnings.String(), "line 2 skipped") {
				t.Errorf("warnings = %q", warnings.String())
			}
		})
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestReadMeasurementsReturnsReadError(t *testing.T) {
	want := errors.New("read failed")
	if _, err := ReadMeasurements(failingReader{want}, &bytes.Buffer{}); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func TestMedian(t *testing.T) {
	numbers := []float64{3, 1, 2}
	assertClose(t, "odd", median(numbers), 2)
	assertClose(t, "even", median([]float64{4, 1, 3, 2}), 2.5)
	if numbers[0] != 3 {
		t.Error("median reordered its input")
	}
}

func TestTCritical(t *testing.T) {
	assertClose(t, "df 4", tCritical(4), 2.776)
	assertClose(t, "df 30", tCritical(30), 2.042)
	assertClose(t, "df 31", tCritical(31), 1.96)
}

func TestCalculateMatchesSpreadsheetFit(t *testing.T) {
	report := mustCalculate(t, sampleMeasurements())
	assertClose(t, "slope", report.Slope, 0.99425)
	assertClose(t, "intercept", report.Intercept, -0.0725)
	assertClose(t, "contour comp", report.ContourComp, 0.03625)
	if math.Abs(report.SlopeRange.Low-0.99214) > 1e-4 || math.Abs(report.SlopeRange.High-0.99636) > 1e-4 {
		t.Errorf("slope range = %+v", report.SlopeRange)
	}
	if len(report.Outliers) != 0 {
		t.Errorf("outliers = %+v", report.Outliers)
	}
}

// The old ratio-based calculation reported a pure offset as shrinkage, and gave
// these two opposite cases the same shape of answer.
func TestCalculateSeparatesShrinkageFromOffset(t *testing.T) {
	sizes := []float64{20, 30, 40, 60, 80}

	offset := mustCalculate(t, exactMeasurements(1, -0.2, sizes...))
	assertClose(t, "offset slope", offset.Slope, 1)
	assertClose(t, "offset intercept", offset.Intercept, -0.2)
	assertClose(t, "offset contour comp", offset.ContourComp, 0.1)

	shrinkage := mustCalculate(t, exactMeasurements(0.99, 0, sizes...))
	assertClose(t, "shrinkage slope", shrinkage.Slope, 0.99)
	assertClose(t, "shrinkage intercept", shrinkage.Intercept, 0)
	assertClose(t, "scale factor", shrinkage.ScaleFactor(), 1/0.99)
}

func TestCalculateLeavesOutTypo(t *testing.T) {
	measurements := sampleMeasurements()
	measurements[5].Actual = 37.91

	report := mustCalculate(t, measurements)
	if len(report.Outliers) != 1 || report.Outliers[0].Line != 6 {
		t.Fatalf("outliers = %+v", report.Outliers)
	}
	if report.PointsUsed != 5 {
		t.Errorf("points used = %d, want 5", report.PointsUsed)
	}
	if math.Abs(report.Slope-0.99425) > 0.002 {
		t.Errorf("slope = %v, the typo moved the fit", report.Slope)
	}
}

func TestCalculateKeepsPointsNearTheFloor(t *testing.T) {
	// Exact points leave a robust deviation of zero, so only the floor stops
	// these 0.03 mm readings being called outliers.
	measurements := exactMeasurements(1, 0, 20, 30, 40, 50, 60, 70)
	measurements[2].Actual += 0.03
	if report := mustCalculate(t, measurements); len(report.Outliers) != 0 {
		t.Errorf("outliers = %+v", report.Outliers)
	}
}

func TestCalculateWarnsAboutNarrowRange(t *testing.T) {
	report := mustCalculate(t, []*Measurement{
		{Expected: 20, Actual: 19.85},
		{Expected: 20, Actual: 19.86},
		{Expected: 21, Actual: 20.82},
		{Expected: 21, Actual: 20.83},
	})
	if !hasWarning(report, "uncertain") {
		t.Errorf("warnings = %q", report.Warnings)
	}
	if !hasWarning(report, "2 sizes") {
		t.Errorf("warnings = %q", report.Warnings)
	}
}

func TestCalculateWarnsAboutFewPoints(t *testing.T) {
	report := mustCalculate(t, exactMeasurements(0.99, 0, 20, 40, 60))
	if !hasWarning(report, "only 3 points") {
		t.Errorf("warnings = %q", report.Warnings)
	}
}

func TestCalculateWarnsWhenOffsetIsNoise(t *testing.T) {
	report := mustCalculate(t, []*Measurement{
		{Expected: 20, Actual: 19.81},
		{Expected: 20, Actual: 19.79},
		{Expected: 100, Actual: 99.01},
		{Expected: 100, Actual: 98.99},
	})
	if !hasWarning(report, "told apart from zero") {
		t.Errorf("warnings = %q", report.Warnings)
	}
}

func TestCalculateCleanWideRangeHasNoWarnings(t *testing.T) {
	report := mustCalculate(t, exactMeasurements(0.995, -0.1, 20, 20, 60, 60, 100, 100))
	if len(report.Warnings) != 0 {
		t.Errorf("warnings = %q", report.Warnings)
	}
}

func TestCalculateRejectsUnfittableData(t *testing.T) {
	cases := map[string][]*Measurement{
		"same size":       exactMeasurements(0.99, 0, 20, 20, 20, 20),
		"two points":      exactMeasurements(0.99, 0, 20, 40),
		"only holes":      {{Expected: 20, Actual: 19.8, Feature: FeatureHole}},
		"no measurements": nil,
	}
	for name, measurements := range cases {
		t.Run(name, func(t *testing.T) {
			if report, err := Calculate(measurements); err == nil {
				t.Errorf("expected an error, got %+v", report)
			}
		})
	}
}

func TestCalculateHolesAreMeasuredSeparately(t *testing.T) {
	measurements := exactMeasurements(0.99, -0.1, 20, 40, 60, 80)
	// Outer contours are small, so a sign flip of their offset would say the
	// holes are too large. They are in fact 0.2 mm small after scaling.
	measurements = append(measurements,
		&Measurement{Expected: 10, Actual: 0.99*10 - 0.2, Feature: FeatureHole},
		&Measurement{Expected: 20, Actual: 0.99*20 - 0.2, Feature: FeatureHole},
	)

	report := mustCalculate(t, measurements)
	assertClose(t, "slope unaffected by holes", report.Slope, 0.99)
	assertClose(t, "hole comp", report.HoleComp, 0.1)
	if report.HolesMeasured != 2 {
		t.Errorf("holes measured = %d, want 2", report.HolesMeasured)
	}
}

func TestCalculateFitsEachAxis(t *testing.T) {
	x := exactMeasurements(0.99, 0, 20, 40, 60)
	y := exactMeasurements(0.995, 0, 20, 40, 60)
	for _, m := range x {
		m.Feature = FeatureX
	}
	for _, m := range y {
		m.Feature = FeatureY
	}

	report := mustCalculate(t, append(x, y...))
	if len(report.Axes) != 2 {
		t.Fatalf("axes = %+v", report.Axes)
	}
	assertClose(t, "x slope", report.Axes[0].Line.Slope, 0.99)
	assertClose(t, "y slope", report.Axes[1].Line.Slope, 0.995)
}
