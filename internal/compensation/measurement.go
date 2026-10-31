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
		return 0, newMessage(CodeNotANumber, field)
	}
	if value <= 0 {
		return 0, newMessage(CodeNotPositive, field)
	}
	return value, nil
}

func parseFeature(field string) (Feature, error) {
	switch feature := Feature(strings.ToLower(field)); feature {
	case FeatureX, FeatureY, FeatureHole:
		return feature, nil
	default:
		return "", newMessage(CodeUnknownLabel, field)
	}
}

// NewMeasurement validates one measurement given as text. Number is the line or
// row it came from, which reports use to point back at it. An empty feature is
// an outer dimension. Any error is a Message.
func NewMeasurement(number int, expected, actual, feature string) (*Measurement, error) {
	expectedSize, err := parseSize(expected)
	if err != nil {
		return nil, err
	}
	actualSize, err := parseSize(actual)
	if err != nil {
		return nil, err
	}
	parsedFeature := FeatureOuter
	if feature != "" {
		if parsedFeature, err = parseFeature(feature); err != nil {
			return nil, err
		}
	}
	return &Measurement{Line: number, Expected: expectedSize, Actual: actualSize, Feature: parsedFeature}, nil
}

func parseLine(number int, fields []string) (*Measurement, error) {
	switch len(fields) {
	case 2:
		return NewMeasurement(number, fields[0], fields[1], "")
	case 3:
		return NewMeasurement(number, fields[0], fields[1], fields[2])
	default:
		return nil, newMessage(CodeFieldCount, len(fields))
	}
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
