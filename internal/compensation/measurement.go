// Package compensation calculates dimensional compensation for 3D printing
// from measurements of test prints.
package compensation

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Feature says what a measurement was taken of.
type Feature string

// The features a measurement line can be labelled with. An unlabelled line is
// an outer dimension on an unspecified axis.
const (
	FeatureOuter Feature = ""
	FeatureX     Feature = "x"
	FeatureY     Feature = "y"
	FeatureHole  Feature = "hole"
)

// Measurement is one printed dimension compared with the size it should be.
type Measurement struct {
	Line     int
	Expected float64
	Actual   float64
	Feature  Feature
}

func parseSize(field string) (float64, error) {
	// A decimal comma is what a Finnish keyboard or spreadsheet produces.
	value, err := strconv.ParseFloat(strings.ReplaceAll(field, ",", "."), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%q is not a number", field)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%q is not a positive size", field)
	}
	return value, nil
}

func parseFeature(field string) (Feature, error) {
	switch feature := Feature(strings.ToLower(field)); feature {
	case FeatureX, FeatureY, FeatureHole:
		return feature, nil
	default:
		return "", fmt.Errorf("unknown label %q, expected x, y or hole", field)
	}
}

func parseLine(number int, fields []string) (*Measurement, error) {
	if len(fields) != 2 && len(fields) != 3 {
		return nil, fmt.Errorf("expected \"expected actual [x|y|hole]\", got %d fields", len(fields))
	}
	expected, err := parseSize(fields[0])
	if err != nil {
		return nil, err
	}
	actual, err := parseSize(fields[1])
	if err != nil {
		return nil, err
	}
	feature := FeatureOuter
	if len(fields) == 3 {
		if feature, err = parseFeature(fields[2]); err != nil {
			return nil, err
		}
	}
	return &Measurement{Line: number, Expected: expected, Actual: actual, Feature: feature}, nil
}

// ReadMeasurements parses lines of "expected actual [x|y|hole]" from r, in any
// whitespace. Blank lines and lines starting with # are ignored. Lines that
// cannot be parsed are skipped and reported on warnings, so one bad line does
// not lose the rest of a measuring session.
func ReadMeasurements(r io.Reader, warnings io.Writer) ([]*Measurement, error) {
	var measurements []*Measurement

	scanner := bufio.NewScanner(r)
	for number := 1; scanner.Scan(); number++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		measurement, err := parseLine(number, strings.Fields(line))
		if err != nil {
			fmt.Fprintf(warnings, "line %d skipped: %v\n", number, err)
			continue
		}
		measurements = append(measurements, measurement)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return measurements, nil
}
