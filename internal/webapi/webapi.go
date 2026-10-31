// Package webapi is the JSON interface between the web page and the
// calculation. It lives apart from the WebAssembly entry point so it can be
// tested as ordinary Go.
package webapi

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/rmkhl/prncompcalc/internal/compensation"
)

// Row is one row of the page's measurement table, as typed.
type Row struct {
	Row      int    `json:"row"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Feature  string `json:"feature"`
}

// Issue is a translatable message. Row is 0 when it is not about a single row.
type Issue struct {
	Row  int               `json:"row,omitempty"`
	Code compensation.Code `json:"code"`
	Args []any             `json:"args,omitempty"`
}

// Range is a 95% confidence interval.
type Range struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

// Axis is the fit of one axis. Shrinkage is a percentage.
type Axis struct {
	Feature   compensation.Feature `json:"feature"`
	Shrinkage float64              `json:"shrinkage"`
	Offset    float64              `json:"offset"`
}

// Outlier is a row left out of the fit.
type Outlier struct {
	Row      int     `json:"row"`
	Residual float64 `json:"residual"`
}

// Result is a successful calculation. Shrinkage and scale are percentages, the
// way OrcaSlicer takes them.
type Result struct {
	PointsUsed     int       `json:"pointsUsed"`
	Sizes          int       `json:"sizes"`
	MinSize        float64   `json:"minSize"`
	MaxSize        float64   `json:"maxSize"`
	Shrinkage      float64   `json:"shrinkage"`
	ShrinkageRange Range     `json:"shrinkageRange"`
	ModelScale     float64   `json:"modelScale"`
	Offset         float64   `json:"offset"`
	OffsetRange    Range     `json:"offsetRange"`
	Scatter        float64   `json:"scatter"`
	ContourComp    float64   `json:"contourComp"`
	HoleComp       *float64  `json:"holeComp"`
	HolesMeasured  int       `json:"holesMeasured"`
	Axes           []Axis    `json:"axes"`
	Outliers       []Outlier `json:"outliers"`
	Warnings       []Issue   `json:"warnings"`
}

// Response is what the page gets back. Exactly one of Result and Error is set
// unless no row was filled in, and RowIssues can accompany either.
type Response struct {
	Result    *Result `json:"result,omitempty"`
	Error     *Issue  `json:"error,omitempty"`
	RowIssues []Issue `json:"rowIssues"`
}

func issueFrom(row int, err error) Issue {
	var message compensation.Message
	if errors.As(err, &message) {
		return Issue{Row: row, Code: message.Code, Args: message.Args}
	}
	// Everything the calculation returns is a Message, so this is a bug. It
	// still has to reach the page as something it can show.
	return Issue{Row: row, Code: "internal", Args: []any{err.Error()}}
}

func isBlank(row Row) bool {
	return strings.TrimSpace(row.Expected) == "" && strings.TrimSpace(row.Actual) == ""
}

func toResult(report *compensation.Report) *Result {
	result := &Result{
		PointsUsed:     report.PointsUsed,
		Sizes:          report.Sizes,
		MinSize:        report.MinSize,
		MaxSize:        report.MaxSize,
		Shrinkage:      report.Slope * 100,
		ShrinkageRange: Range{report.SlopeRange.Low * 100, report.SlopeRange.High * 100},
		ModelScale:     report.ScaleFactor() * 100,
		Offset:         report.Intercept,
		OffsetRange:    Range{report.InterceptRange.Low, report.InterceptRange.High},
		Scatter:        report.ResidualScatter,
		ContourComp:    report.ContourComp,
		HolesMeasured:  report.HolesMeasured,
		Axes:           []Axis{},
		Outliers:       []Outlier{},
		Warnings:       []Issue{},
	}
	if report.HolesMeasured > 0 {
		holeComp := report.HoleComp
		result.HoleComp = &holeComp
	}
	for _, axis := range report.Axes {
		result.Axes = append(result.Axes, Axis{axis.Feature, axis.Line.Slope * 100, axis.Line.Intercept})
	}
	for _, outlier := range report.Outliers {
		result.Outliers = append(result.Outliers, Outlier{outlier.Line, outlier.Residual})
	}
	for _, warning := range report.Warnings {
		result.Warnings = append(result.Warnings, Issue{Code: warning.Code, Args: warning.Args})
	}
	return result
}

// Calculate turns table rows into a report. Rows left blank are ignored and bad
// rows are reported while the good ones are still used, because the page
// recalculates on every keystroke and a half-typed row must not blank the
// result.
func Calculate(rows []Row) Response {
	response := Response{RowIssues: []Issue{}}

	measurements := make([]*compensation.Measurement, 0, len(rows))
	for _, row := range rows {
		if isBlank(row) {
			continue
		}
		measurement, err := compensation.NewMeasurement(row.Row, row.Expected, row.Actual, row.Feature)
		if err != nil {
			response.RowIssues = append(response.RowIssues, issueFrom(row.Row, err))
			continue
		}
		measurements = append(measurements, measurement)
	}
	if len(measurements) == 0 {
		return response
	}

	report, err := compensation.Calculate(measurements)
	if err != nil {
		issue := issueFrom(0, err)
		response.Error = &issue
		return response
	}
	response.Result = toResult(report)
	return response
}

// CalculateJSON is Calculate over JSON strings, for the WebAssembly boundary.
func CalculateJSON(request string) (string, error) {
	var rows []Row
	if err := json.Unmarshal([]byte(request), &rows); err != nil {
		return "", err
	}
	response, err := json.Marshal(Calculate(rows))
	if err != nil {
		return "", err
	}
	return string(response), nil
}
