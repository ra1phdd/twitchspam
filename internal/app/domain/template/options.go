package template

import (
	"twitchspam/internal/app/infrastructure/config"
)

type OptionsTemplate struct{}

func NewOptions() *OptionsTemplate {
	return &OptionsTemplate{}
}

func (ot *OptionsTemplate) parse(words *[]string) (bool, config.Options) {
	opts := config.Options{}
	clean := (*words)[:0]

	isRegex := false
	for _, w := range *words {
		switch w {
		case "-regex":
			isRegex = true
		case "-first":
			*opts.IsFirst = true
		case "-nosub":
			*opts.NoSub = true
		case "-novip":
			*opts.NoVip = true
		case "-norepeat":
			*opts.NoRepeat = true
		case "-oneword":
			*opts.OneWord = true
		case "-contains":
			*opts.Contains = true
		case "-sub", "-vip", "-nocontains":
			continue
		default:
			clean = append(clean, w)
		}
	}

	*words = clean
	return isRegex, opts
}

func (ot *OptionsTemplate) merge(dst, src *config.Options) {
	if src.IsFirst != nil {
		dst.IsFirst = src.IsFirst
	}
	if src.NoSub != nil {
		dst.NoSub = src.NoSub
	}
	if src.NoVip != nil {
		dst.NoVip = src.NoVip
	}
	if src.NoRepeat != nil {
		dst.NoRepeat = src.NoRepeat
	}
	if src.OneWord != nil {
		dst.OneWord = src.OneWord
	}
	if src.Contains != nil {
		dst.Contains = src.Contains
	}
}
