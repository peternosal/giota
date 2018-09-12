package units

import (
	"math"
	"strconv"
)

type Unit float64

var (
	I  = Unit(1)
	Ki = Unit(math.Pow10(3))
	Mi = Unit(math.Pow10(6))
	Gi = Unit(math.Pow10(9))
	Ti = Unit(math.Pow10(12))
	Pi = Unit(math.Pow10(15))
)

func ConvertUnits(val float64, from Unit, to Unit) float64 {
	value := Unit(val)
	// convert to I unit by multiplying with the current unit
	value *= from
	// convert to the target unit by dividing by it
	if to == I {
		return math.Round(float64(value))
	}
	value /= to
	return float64(value)
}

func ConvertUnitString(value string, from Unit, to Unit) (float64, error) {
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return ConvertUnits(floatValue, from, to), nil
}
