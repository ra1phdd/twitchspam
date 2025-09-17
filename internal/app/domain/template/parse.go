package template

import (
	"math"
	"strconv"
)

type ParserTemplate struct{}

func NewParser() *ParserTemplate {
	return &ParserTemplate{}
}

func (p *ParserTemplate) parseIntArg(valStr string, min, max int) (int, bool) {
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, false
	}
	if (min != -1 && val < min) || (max != -1 && val > max) {
		return 0, false
	}
	return val, true
}

func (p *ParserTemplate) parseFloatArg(valStr string, min, max float64) (float64, bool) {
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil || val < min || val > max {
		return 0, false
	}

	return math.Round(val*100) / 100, true
}
