package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Measurement struct {
	expected   float64
	actual     float64
	shrinkage  float64
	adjustment float64
}

func newMeasurement(expected, actual float64) *Measurement {
	return &Measurement{
		expected:   expected,
		actual:     actual,
		shrinkage:  actual / expected,
		adjustment: expected - actual,
	}
}

func readMeasurements(f *os.File) []*Measurement {
	// Read the measurements from the file
	var measurements []*Measurement

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Parse the line
		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			fmt.Println("Invalid line: ", line)
		} else {
			expected, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				fmt.Println("Error parsing expected value: ", parts[0])
				continue
			}
			actual, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				fmt.Println("Error parsing actual value: ", parts[1])
				continue
			}
			m := newMeasurement(expected, actual)
			measurements = append(measurements, m)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return measurements
}

func calculateMean(numbers []float64) float64 {
	var sum float64
	for _, number := range numbers {
		sum += number
	}
	return sum / float64(len(numbers))
}

func calculateMedian(numbers []float64) float64 {
	sort.Float64s(numbers)
	middle := len(numbers) / 2
	if len(numbers)%2 == 0 {
		return (numbers[middle-1] + numbers[middle]) / 2
	} else {
		return numbers[middle]
	}
}

func calculateDeviation(number float64, numbers []float64) float64 {
	var deviation float64
	for _, num := range numbers {
		deviation += math.Abs(num - number)
	}
	return deviation
}

func findLeastDeviation(numbers []float64) float64 {
	mean := calculateMean(numbers)
	median := calculateMedian(numbers)

	meanDeviation := calculateDeviation(mean, numbers)
	medianDeviation := calculateDeviation(median, numbers)

	if meanDeviation < medianDeviation {
		return mean
	} else {
		return median
	}
}

func simulateMeasuring(measurements []*Measurement, shrinkage float64) []*Measurement {
	var simulatedMeasurements []*Measurement
	for _, measurement := range measurements {
		simulatedMeasurements = append(simulatedMeasurements, newMeasurement(measurement.expected, measurement.expected*shrinkage))
	}
	return simulatedMeasurements
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ", os.Args[0], " <file>")
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	measurements := readMeasurements(f)

	// Extract the shrinkages and find the least deviation
	var shrinkages []float64
	for _, measurement := range measurements {
		shrinkages = append(shrinkages, measurement.shrinkage)
	}
	leastDeviatingShrinkage := findLeastDeviation(shrinkages)

	// Extract the adjustments and find the least deviation
	var adjustments []float64
	for _, measurement := range measurements {
		adjustments = append(adjustments, measurement.adjustment)
	}
	leastDeviatingAdjustment := findLeastDeviation(adjustments)

	simulatedMeasurements := simulateMeasuring(measurements, leastDeviatingShrinkage)
	var simulatedAdjustments []float64
	for _, measurement := range simulatedMeasurements {
		simulatedAdjustments = append(simulatedAdjustments, measurement.adjustment)
	}
	leastDeviatingSimulatedAdjustment := findLeastDeviation(simulatedAdjustments)

	fmt.Printf("Shrinkage: %.4f, Adjustment: %.4f, Simulated adjustment %.4f\n", leastDeviatingShrinkage, leastDeviatingAdjustment, leastDeviatingSimulatedAdjustment)
}
