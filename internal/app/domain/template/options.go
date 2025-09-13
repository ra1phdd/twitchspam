package template

import (
	"twitchspam/internal/app/infrastructure/config"
)

type OptionsTemplate struct{}

func NewOptions() *OptionsTemplate {
	return &OptionsTemplate{}
}

var SpamOptions = map[string]struct{}{
	"-regex": {}, "-first": {}, "-nosub": {}, "-novip": {},
	"-norepeat": {}, "-repeat": {}, "-oneword": {}, "-nofirst": {},
	"-contains": {}, "-nocontains": {}, "-vip": {}, "-sub": {},
}

var TimersOptions = map[string]struct{}{
	"-a": {}, "-noa": {}, "-online": {}, "-always": {},
}

func (ot *OptionsTemplate) parseAll(words *[]string, opts map[string]struct{}) map[string]bool {
	clean := (*words)[:0]

	founds := make(map[string]bool)
	for _, w := range *words {
		if _, ok := opts[w]; ok {
			founds[w] = true
			continue
		}

		clean = append(clean, w)
	}

	*words = clean
	return founds
}

func (ot *OptionsTemplate) parse(words *[]string, opt string) *bool {
	clean := (*words)[:0]

	var foundOpt *bool
	for _, w := range *words {
		if w == opt {
			*foundOpt = true
			break
		}

		clean = append(clean, w)

	}

	*words = clean
	return foundOpt
}

func (ot *OptionsTemplate) merge(dst *config.SpamOptions, src map[string]bool) {
	if dst == nil {
		return
	}

	if _, ok := src["-nofirst"]; ok {
		dst.IsFirst = false
	}

	if _, ok := src["-first"]; ok {
		dst.IsFirst = true
	}

	if _, ok := src["-nosub"]; ok {
		dst.NoSub = true
	}

	if _, ok := src["-sub"]; ok {
		dst.NoSub = false
	}

	if _, ok := src["-novip"]; ok {
		dst.NoVip = true
	}

	if _, ok := src["-vip"]; ok {
		dst.NoVip = false
	}

	if _, ok := src["-norepeat"]; ok {
		dst.NoRepeat = true
	}

	if _, ok := src["-repeat"]; ok {
		dst.NoRepeat = false
	}

	if _, ok := src["-oneword"]; ok {
		dst.OneWord = true
	}

	if _, ok := src["-noontains"]; ok {
		dst.Contains = false
	}

	if _, ok := src["-contains"]; ok {
		dst.Contains = true
	}
}
