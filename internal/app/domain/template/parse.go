package template

import (
	"math"
	"strconv"
)

type ParserTemplate struct{}

func NewParser() *ParserTemplate {
	return &ParserTemplate{}
}

func (p *ParserTemplate) ParseIntArg(valStr string, minVal, maxVal int) (int, bool) {
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, false
	}
	if (minVal != -1 && val < minVal) || (maxVal != -1 && val > maxVal) {
		return 0, false
	}
	return val, true
}

func (p *ParserTemplate) ParseFloatArg(valStr string, minVal, maxVal float64) (float64, bool) {
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil || val < minVal || val > maxVal {
		return 0, false
	}

	return math.Round(val*100) / 100, true
}
