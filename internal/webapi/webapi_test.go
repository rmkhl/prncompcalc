package webapi

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/rmkhl/prncompcalc/internal/compensation"
)

func sampleRows() []Row {
	return []Row{
		{Row: 1, Expected: "20", Actual: "19,8"},
		{Row: 2, Expected: "20", Actual: "19.83"},
		{Row: 3, Expected: "30", Actual: "29.74"},
		{Row: 4, Expected: "30", Actual: "29.76"},
		{Row: 5, Expected: "40", Actual: "39.69"},
		{Row: 6, Expected: "40", Actual: "39.71"},
	}
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestCalculateReportsPercentages(t *testing.T) {
	response := Calculate(sampleRows())
	if response.Result == nil {
		t.Fatalf("no result: %+v", response)
	}
	assertClose(t, "shrinkage", response.Result.Shrinkage, 99.425)
	assertClose(t, "offset", response.Result.Offset, -0.0725)
	assertClose(t, "contour comp", response.Result.ContourComp, 0.03625)
	if response.Result.HoleComp != nil {
		t.Errorf("hole comp = %v, want none", *response.Result.HoleComp)
	}
}

func TestCalculateIgnoresBlankRows(t *testing.T) {
	rows := append(sampleRows(), Row{Row: 7, Expected: " ", Actual: "", Feature: "x"})
	if response := Calculate(rows); len(response.RowIssues) != 0 || response.Result == nil {
		t.Errorf("response = %+v", response)
	}
}

func TestCalculateKeepsResultWhileARowIsHalfTyped(t *testing.T) {
	rows := append(sampleRows(), Row{Row: 7, Expected: "50", Actual: ""})
	response := Calculate(rows)
	if response.Result == nil {
		t.Fatalf("no result: %+v", response)
	}
	if len(response.RowIssues) != 1 || response.RowIssues[0].Row != 7 {
		t.Errorf("row issues = %+v", response.RowIssues)
	}
}

func TestCalculateNamesOutlierRow(t *testing.T) {
	rows := sampleRows()
	rows[5].Actual = "37.91"
	response := Calculate(rows)
	if response.Result == nil || len(response.Result.Outliers) != 1 || response.Result.Outliers[0].Row != 6 {
		t.Errorf("response = %+v", response)
	}
}

func TestCalculateReturnsErrorCode(t *testing.T) {
	response := Calculate([]Row{
		{Row: 1, Expected: "20", Actual: "19.8"},
		{Row: 2, Expected: "20", Actual: "19.9"},
		{Row: 3, Expected: "20", Actual: "19.7"},
	})
	if response.Error == nil || response.Error.Code != compensation.CodeOneSize {
		t.Errorf("response = %+v", response)
	}
}

func TestCalculateWithNothingEnteredIsNotAnError(t *testing.T) {
	response := Calculate([]Row{{Row: 1}})
	if response.Error != nil || response.Result != nil {
		t.Errorf("response = %+v", response)
	}
}

func TestCalculateHoleComp(t *testing.T) {
	rows := append(sampleRows(), Row{Row: 7, Expected: "10", Actual: "9.7", Feature: "hole"})
	response := Calculate(rows)
	if response.Result == nil || response.Result.HoleComp == nil {
		t.Fatalf("response = %+v", response)
	}
	if response.Result.HolesMeasured != 1 {
		t.Errorf("holes measured = %d", response.Result.HolesMeasured)
	}
}

func TestCalculateJSONRoundTrip(t *testing.T) {
	request, err := json.Marshal(sampleRows())
	if err != nil {
		t.Fatal(err)
	}
	got, err := CalculateJSON(string(request))
	if err != nil {
		t.Fatal(err)
	}
	var response Response
	if err := json.Unmarshal([]byte(got), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result == nil {
		t.Errorf("response = %s", got)
	}
}

func TestCalculateJSONRejectsBadRequest(t *testing.T) {
	if _, err := CalculateJSON("not json"); err == nil {
		t.Error("expected an error")
	}
}

// The page has its own wording for every code, in both languages, and nothing
// else would notice one going missing.
func TestPageTranslatesEveryCode(t *testing.T) {
	source, err := os.ReadFile("../../web/i18n.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range compensation.Codes() {
		if got := strings.Count(string(source), string(code)+":"); got != 2 {
			t.Errorf("%s appears %d times in web/i18n.js, want one per language", code, got)
		}
	}
}
