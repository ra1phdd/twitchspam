package template

import (
	"strings"
	"twitchspam/internal/app/infrastructure/config"
)

type OptionsTemplate struct{}

func NewOptions() *OptionsTemplate {
	return &OptionsTemplate{}
}

var SpamExceptOptions = map[string]struct{}{
	"-regex":  {},
	"-repeat": {}, "-norepeat": {},
	"-oneword": {}, "-nooneword": {},
	"-contains": {}, "-nocontains": {},
	"-case": {}, "-nocase": {},
}

var MwordOptions = map[string]struct{}{
	"-regex": {},
	"-first": {}, "-nofirst": {},
	"-sub": {}, "-nosub": {},
	"-vip": {}, "-novip": {},
	"-repeat": {}, "-norepeat": {},
	"-oneword": {}, "-nooneword": {},
	"-contains": {}, "-nocontains": {},
	"-case": {}, "-nocase": {},
}

var TimersOptions = map[string]struct{}{
	"-a": {}, "-noa": {},
	"-online": {}, "-always": {},
}

func (ot *OptionsTemplate) ParseAll(words *[]string, opts map[string]struct{}) map[string]bool {
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

func (ot *OptionsTemplate) Parse(words *[]string, opt string) *bool {
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

func (ot *OptionsTemplate) MergeMword(dst config.MwordOptions, src map[string]bool) config.MwordOptions {
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

	if _, ok := src["-nooneword"]; ok {
		dst.OneWord = false
	}

	if _, ok := src["-oneword"]; ok {
		dst.OneWord = true
	}

	if _, ok := src["-nocontains"]; ok {
		dst.Contains = false
	}

	if _, ok := src["-contains"]; ok {
		dst.Contains = true
	}

	if _, ok := src["-nocase"]; ok {
		dst.CaseSensitive = false
	}

	if _, ok := src["-case"]; ok {
		dst.CaseSensitive = true
	}

	return dst
}

func (ot *OptionsTemplate) MergeExcept(dst config.ExceptOptions, src map[string]bool) config.ExceptOptions {
	if dst == (config.ExceptOptions{}) { // значение по умолчанию
		dst.OneWord = true
		dst.NoVip = true
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

	if _, ok := src["-nooneword"]; ok {
		dst.OneWord = false
	}

	if _, ok := src["-oneword"]; ok {
		dst.OneWord = true
	}

	if _, ok := src["-nocontains"]; ok {
		dst.Contains = false
	}

	if _, ok := src["-contains"]; ok {
		dst.Contains = true
	}

	if _, ok := src["-nocase"]; ok {
		dst.CaseSensitive = false
	}

	if _, ok := src["-case"]; ok {
		dst.CaseSensitive = true
	}

	return dst
}

func (ot *OptionsTemplate) ExceptToString(opts config.ExceptOptions) string {
	result := []string{
		func() string {
			if opts.NoRepeat {
				return "-norepeat"
			}
			return "-repeat"
		}(),
		func() string {
			if opts.OneWord {
				return "-oneword"
			}
			return "-nooneword"
		}(),
		func() string {
			if opts.Contains {
				return "-contains"
			}
			return "-nocontains"
		}(),
		func() string {
			if opts.CaseSensitive {
				return "-case"
			}
			return "-nocase"
		}(),
		func() string {
			if !opts.NoSub {
				return "-sub"
			}
			return "-nosub"
		}(),
		func() string {
			if !opts.NoVip {
				return "-vip"
			}
			return "-novip"
		}(),
	}
	return strings.Join(result, " ")
}

func (ot *OptionsTemplate) MwordToString(opts config.MwordOptions) string {
	result := []string{
		func() string {
			if opts.NoRepeat {
				return "-norepeat"
			}
			return "-repeat"
		}(),
		func() string {
			if opts.OneWord {
				return "-oneword"
			}
			return "-nooneword"
		}(),
		func() string {
			if opts.Contains {
				return "-contains"
			}
			return "-nocontains"
		}(),
		func() string {
			if opts.CaseSensitive {
				return "-case"
			}
			return "-nocase"
		}(),
		func() string {
			if opts.IsFirst {
				return "-first"
			}
			return "-nofirst"
		}(),
		func() string {
			if !opts.NoSub {
				return "-sub"
			}
			return "-nosub"
		}(),
		func() string {
			if !opts.NoVip {
				return "-vip"
			}
			return "-novip"
		}(),
	}
	return strings.Join(result, " ")
}
