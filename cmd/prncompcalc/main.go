// Command prncompcalc calculates 3D-printing dimensional compensation from a
// file of "expected actual" measurements.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rmkhl/prncompcalc/internal/compensation"
	"github.com/rmkhl/prncompcalc/internal/version"
)

func calculateFile(path string, warnings io.Writer) (*compensation.Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	measurements, err := compensation.ReadMeasurements(f, warnings)
	if err != nil {
		return nil, err
	}
	if len(measurements) == 0 {
		return nil, errors.New("no valid measurements found")
	}
	return compensation.Calculate(measurements)
}

func writeReport(w io.Writer, r *compensation.Report) {
	fmt.Fprintf(w, "Points used:   %d, %d sizes from %.2f to %.2f mm\n", r.PointsUsed, r.Sizes, r.MinSize, r.MaxSize)
	fmt.Fprintf(w, "Shrinkage:     %.5f (95%% range %.5f to %.5f)\n", r.Slope, r.SlopeRange.Low, r.SlopeRange.High)
	fmt.Fprintf(w, "Model scale:   %.5f\n", r.ScaleFactor())
	fmt.Fprintf(w, "Offset:        %+.4f mm (95%% range %+.4f to %+.4f mm)\n", r.Intercept, r.InterceptRange.Low, r.InterceptRange.High)
	fmt.Fprintf(w, "Scatter:       %.4f mm\n", r.ResidualScatter)
	fmt.Fprintf(w, "Contour comp:  %+.4f mm per side\n", r.ContourComp)
	if r.HolesMeasured > 0 {
		fmt.Fprintf(w, "Hole comp:     %+.4f mm per side (%d measured)\n", r.HoleComp, r.HolesMeasured)
	} else {
		fmt.Fprintln(w, "Hole comp:     not measured, add lines labelled hole")
	}
	for _, axis := range r.Axes {
		fmt.Fprintf(w, "Axis %s:        shrinkage %.5f, offset %+.4f mm\n", strings.ToUpper(string(axis.Feature)), axis.Line.Slope, axis.Line.Intercept)
	}
	for _, outlier := range r.Outliers {
		fmt.Fprintf(w, "Outlier:       line %d left out, %g measured as %g is %+.3f mm from the fit\n", outlier.Line, outlier.Expected, outlier.Actual, outlier.Residual)
	}
	for _, warning := range r.Warnings {
		fmt.Fprintf(w, "Warning:       %s\n", warning)
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("prncompcalc", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: prncompcalc [-version] <file>")
		flags.PrintDefaults()
	}
	showVersion := flags.Bool("version", false, "print the version and exit")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version.Version)
		return 0
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}

	report, err := calculateFile(flags.Arg(0), stderr)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	writeReport(stdout, report)
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
