package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rmkhl/prncompcalc/internal/version"
)

const sampleData = "20 19.8\n20 19.83\n30 29.74\n30 29.76\n40 39.69\n40 39.71\n"

func writeInput(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "measurements.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runProgram(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestRunReportsCompensation(t *testing.T) {
	code, stdout, stderr := runProgram(writeInput(t, sampleData))
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	for _, want := range []string{
		"Points used:   6, 3 sizes from 20.00 to 40.00 mm\n",
		"Shrinkage:     99.425% (95% range 99.214% to 99.636%)\n",
		"Model scale:   100.578%\n",
		"Offset:        -0.0725 mm (95% range -0.1381 to -0.0069 mm)\n",
		"Contour comp:  +0.0363 mm per side\n",
		"Hole comp:     not measured",
		"Warning:       shrinkage is uncertain by ±0.21 mm per 100 mm",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output is missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunReportsOutlierLine(t *testing.T) {
	data := strings.Replace(sampleData, "39.71", "37.91", 1)
	_, stdout, _ := runProgram(writeInput(t, data))
	if !strings.Contains(stdout, "Outlier:       line 6 left out") {
		t.Errorf("output:\n%s", stdout)
	}
}

func TestRunKeepsWarningsOffStdout(t *testing.T) {
	code, stdout, stderr := runProgram(writeInput(t, sampleData+"bad\n"))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr, "line 7 skipped") {
		t.Errorf("stderr = %q", stderr)
	}
	if strings.Contains(stdout, "skipped") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestRunFailsCleanly(t *testing.T) {
	cases := map[string]string{
		"empty file":   writeInput(t, ""),
		"only invalid": writeInput(t, "bad\n"),
		"one size":     writeInput(t, "20 19.8\n20 19.9\n20 19.7\n"),
		"missing file": filepath.Join(t.TempDir(), "missing.txt"),
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runProgram(path)
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr, "Error:") || stdout != "" {
				t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, _ := runProgram("-version")
	if code != 0 || strings.TrimSpace(stdout) != version.Version {
		t.Errorf("exit code = %d, stdout = %q", code, stdout)
	}
}

func TestRunUsage(t *testing.T) {
	code, _, stderr := runProgram()
	if code != 2 || !strings.Contains(stderr, "Usage:") {
		t.Errorf("exit code = %d, stderr = %q", code, stderr)
	}
}

// One run of the real binary, for what the tests above cannot see: that main
// passes the arguments on and turns the result into the process exit code.
func TestProgramExitCode(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "prncompcalc")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	if out, err := exec.Command(binary, writeInput(t, sampleData)).CombinedOutput(); err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	err := exec.Command(binary, writeInput(t, "")).Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Errorf("err = %v, want exit code 1", err)
	}
}
