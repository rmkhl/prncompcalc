package compensation

import "fmt"

// Code identifies a warning or error. The web page translates these, so they
// are part of its interface: keep web/i18n.js in step with the list.
type Code string

// The warnings and errors the package can produce.
const (
	CodeFieldCount      Code = "field_count"
	CodeNotANumber      Code = "not_a_number"
	CodeNotPositive     Code = "not_positive"
	CodeUnknownLabel    Code = "unknown_label"
	CodeNoMeasurements  Code = "no_measurements"
	CodeNoOuter         Code = "no_outer"
	CodeTooFewToFit     Code = "too_few_to_fit"
	CodeOneSize         Code = "one_size"
	CodeFewPoints       Code = "few_points"
	CodeFewSizes        Code = "few_sizes"
	CodeUncertainShrink Code = "uncertain_shrinkage"
	CodeOffsetIsNoise   Code = "offset_is_noise"
)

var englishText = map[Code]string{
	CodeFieldCount:      "expected \"expected actual [x|y|hole]\", got %d fields",
	CodeNotANumber:      "%q is not a number",
	CodeNotPositive:     "%q is not a positive size",
	CodeUnknownLabel:    "unknown label %q, expected x, y or hole",
	CodeNoMeasurements:  "no valid measurements found",
	CodeNoOuter:         "no outer measurements to fit",
	CodeTooFewToFit:     "at least %d measurements are needed",
	CodeOneSize:         "at least 2 different sizes are needed, they are what separates shrinkage from offset",
	CodeFewPoints:       "only %d points used, measure at least %d",
	CodeFewSizes:        "only %d sizes measured, %d are needed to show the error is linear",
	CodeUncertainShrink: "shrinkage is uncertain by ±%.2f mm per 100 mm, measure a wider range of sizes or more points",
	CodeOffsetIsNoise:   "the offset cannot be told apart from zero, so the contour compensation may be noise",
}

// Message is a warning or error, kept as a code and arguments so that each
// front end can word it in its own language. It can be used as an error.
//
//nolint:errname // most Messages are warnings, so an Error suffix would mislead
type Message struct {
	Code Code
	Args []any
}

func (m Message) String() string {
	return fmt.Sprintf(englishText[m.Code], m.Args...)
}

func (m Message) Error() string {
	return m.String()
}

// Codes lists every code.
func Codes() []Code {
	codes := make([]Code, 0, len(englishText))
	for code := range englishText {
		codes = append(codes, code)
	}
	return codes
}

func newMessage(code Code, args ...any) Message {
	return Message{code, args}
}
