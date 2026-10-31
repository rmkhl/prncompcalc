# prncompcalc

Calculates dimensional compensation for 3D printing from a set of test-print
measurements: how far the printed sizes are from the sizes they should be, as a
shrinkage ratio and as a fixed adjustment.

## Getting started

Everything goes through `make`. `make prepare` installs Go and golangci-lint
into `.go/` inside the project, so nothing is installed on the system.

```bash
make prepare   # install the local toolchain
make all       # build bin/prncompcalc
make test      # run the tests
make help      # list every target
```

## Usage

Give it a file with one measurement per line: the expected size, the measured
size in millimetres, and optionally a label. Any whitespace separates the
fields, a decimal comma works, and blank lines and lines starting with `#` are
ignored.

```text
# size  measured  label
20 19.80 x
20 19.83 y
60 59.57 x
60 59.60 y
100 99.35 x
100 99.37 y
10 9.71 hole
```

| Label | Meaning |
| ----- | ------- |
| none | Outer dimension |
| `x`, `y` | Outer dimension on that axis, which also gets its own fit |
| `hole` | Hole diameter, used only for the hole compensation |

```console
$ bin/prncompcalc measurements.txt
Points used:   6, 3 sizes from 20.00 to 100.00 mm
Shrinkage:     0.99431 (95% range 0.99373 to 0.99489)
Model scale:   1.00572
Offset:        -0.0721 mm (95% range -0.1115 to -0.0326 mm)
Scatter:       0.0166 mm
Contour comp:  +0.0360 mm per side
Hole comp:     +0.1166 mm per side (1 measured)
Axis X:        shrinkage 0.99437, offset -0.0892 mm
Axis Y:        shrinkage 0.99425, offset -0.0550 mm
```

| Value | Meaning |
| ----- | ------- |
| Shrinkage | Slope of the fit: printed size as a ratio of model size, as slicers take it |
| Model scale | What to scale a model by to cancel the shrinkage |
| Offset | Constant error of the outer dimension, across both sides |
| Scatter | Typical distance of a measurement from the fit |
| Contour comp | Per-side change to outer contours; positive makes them larger |
| Hole comp | Per-side change to holes; positive makes them larger |

Check the sign convention against your slicer before entering the
compensation values. `bin/prncompcalc -version` prints the version.

## How it works

The model is `measured = shrinkage × size + offset`. Fitting both together is
what separates them: a plain measured/expected ratio counts any offset as extra
shrinkage.

1. A Theil-Sen fit, which a bad value barely moves, finds outliers. A point is
   left out if it is more than 3.5 robust standard deviations from that fit, and
   at least 0.05 mm. The report names its line.
2. Least squares on the remaining points gives the result and its 95% ranges.
3. The ranges decide the warnings. Nothing is judged by the largest residual,
   because that grows as you add measurements.

Hole compensation comes from the holes' own measurements, after the shrinkage
from the outer fit. It is never derived from outer dimensions, because holes
print small for reasons of their own.

## Measuring well

- Cover a wide range of sizes. The further apart they are, the better shrinkage
  and offset separate.
- Measure at least three sizes, or a non-linear error cannot show up.
- Label the axes if the printer may differ between them.
- Confirm the result on a new print. These figures come from fitting the same
  measurements they are judged against.

## License

GPL-3.0. See [LICENSE](LICENSE).
