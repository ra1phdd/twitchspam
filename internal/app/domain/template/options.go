package template

import (
	"strings"
	"twitchspam/internal/app/infrastructure/config"
)

type OptionsTemplate struct{}

func NewOptions() *OptionsTemplate {
	return &OptionsTemplate{}
}

var ExceptOptions = map[string]struct{}{
	"-sub": {}, "-nosub": {},
	"-vip": {}, "-novip": {},
	"-repeat": {}, "-norepeat": {},
	"-oneword": {}, "-nooneword": {},
	"-contains": {}, "-nocontains": {},
	"-case": {}, "-nocase": {},
}

var MwordOptions = map[string]struct{}{
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
	"-always": {}, "-online": {},
}

var CommandOptions = map[string]struct{}{
	"-private": {}, "-public": {},
	"-always": {}, "-online": {}, "-offline": {},
}

const (
	AlwaysCommandMode = iota
	OnlineCommandMode
	OfflineCommandMode
)

func (ot *OptionsTemplate) ParseAll(input string, opts map[string]struct{}) (string, map[string]bool) {
	words := strings.Fields(input)

	var clean []string
	founds := make(map[string]bool)

	for _, w := range words {
		if _, ok := opts[strings.ToLower(w)]; ok {
			founds[strings.ToLower(w)] = true
			continue
		}
		clean = append(clean, w)
	}

	return strings.Join(clean, " "), founds
}

func (ot *OptionsTemplate) MergeExcept(dst config.ExceptOptions, src map[string]bool, isDefault bool) config.ExceptOptions {
	if isDefault && dst == (config.ExceptOptions{}) { // значение по умолчанию
		dst.OneWord = true
		dst.NoVip = true
	}

	return ot.mergeExcept(dst, src)
}

func (ot *OptionsTemplate) MergeEmoteExcept(dst config.ExceptOptions, src map[string]bool, isDefault bool) config.ExceptOptions {
	if isDefault && dst == (config.ExceptOptions{}) { // значение по умолчанию
		dst.NoVip = true
	}

	return ot.mergeExcept(dst, src)
}

func (ot *OptionsTemplate) mergeExcept(dst config.ExceptOptions, src map[string]bool) config.ExceptOptions {
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

func (ot *OptionsTemplate) MergeTimer(dst config.TimerOptions, src map[string]bool) config.TimerOptions {
	if _, ok := src["-noa"]; ok {
		dst.IsAnnounce = false
	}

	if _, ok := src["-a"]; ok {
		dst.IsAnnounce = true
	}

	if _, ok := src["-online"]; ok {
		dst.IsAlways = false
	}

	if _, ok := src["-always"]; ok {
		dst.IsAlways = true
	}

	return dst
}

func (ot *OptionsTemplate) MergeCommand(dst config.CommandOptions, src map[string]bool) config.CommandOptions {
	if _, ok := src["-public"]; ok {
		dst.IsPrivate = false
	}

	if _, ok := src["-private"]; ok {
		dst.IsPrivate = true
	}

	if _, ok := src["-always"]; ok {
		dst.Mode = AlwaysCommandMode
	}

	if _, ok := src["-online"]; ok {
		dst.Mode = OnlineCommandMode
	}

	if _, ok := src["-offline"]; ok {
		dst.Mode = OfflineCommandMode
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

func (ot *OptionsTemplate) TimerToString(opts config.TimerOptions) string {
	result := []string{
		func() string {
			if opts.IsAnnounce {
				return "-a"
			}
			return "-noa"
		}(),
		func() string {
			if opts.IsAlways {
				return "-always"
			}
			return "-online"
		}(),
	}
	return strings.Join(result, " ")
}

func (ot *OptionsTemplate) CommandToString(opts config.CommandOptions) string {
	result := []string{
		func() string {
			if opts.IsPrivate {
				return "-private"
			}
			return "-public"
		}(),
		func() string {
			switch opts.Mode {
			case OnlineCommandMode:
				return "-online"
			case OfflineCommandMode:
				return "-offline"
			default:
				return "-always"
			}
		}(),
	}
	return strings.Join(result, " ")
}
