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
	"-always": {}, "-online": {}, "-offline": {},
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
	"-always": {}, "-online": {}, "-offline": {},
	"-blue": {}, "-green": {},
	"-orange": {}, "-purple": {},
	"-primary": {},
}

var CommandOptions = map[string]struct{}{
	"-private": {}, "-public": {},
	"-always": {}, "-online": {}, "-offline": {},
}

func (ot *OptionsTemplate) ParseAll(input string, opts map[string]struct{}) (string, map[string]bool) {
	words := strings.Fields(input)

	clean := make([]string, 0, len(words))
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

func (ot *OptionsTemplate) MergeExcept(dst *config.ExceptOptions, src map[string]bool) *config.ExceptOptions {
	return ot.mergeExcept(dst, src)
}

func (ot *OptionsTemplate) MergeEmoteExcept(dst *config.ExceptOptions, src map[string]bool) *config.ExceptOptions {
	return ot.mergeExcept(dst, src)
}

func (ot *OptionsTemplate) mergeExcept(dst *config.ExceptOptions, src map[string]bool) *config.ExceptOptions {
	trueVal, falseVal := true, false
	if dst == nil {
		if len(src) == 0 {
			return nil
		}
		dst = &config.ExceptOptions{}
	}

	if _, ok := src["-nosub"]; ok {
		dst.NoSub = &trueVal
	}

	if _, ok := src["-sub"]; ok {
		dst.NoSub = &falseVal
	}

	if _, ok := src["-novip"]; ok {
		dst.NoVip = &trueVal
	}

	if _, ok := src["-vip"]; ok {
		dst.NoVip = &falseVal
	}

	if _, ok := src["-norepeat"]; ok {
		dst.NoRepeat = &trueVal
	}

	if _, ok := src["-repeat"]; ok {
		dst.NoRepeat = &falseVal
	}

	if _, ok := src["-nooneword"]; ok {
		dst.OneWord = &falseVal
	}

	if _, ok := src["-oneword"]; ok {
		dst.OneWord = &trueVal
	}

	if _, ok := src["-nocontains"]; ok {
		dst.Contains = &falseVal
	}

	if _, ok := src["-contains"]; ok {
		dst.Contains = &trueVal
	}

	if _, ok := src["-nocase"]; ok {
		dst.CaseSensitive = &falseVal
	}

	if _, ok := src["-case"]; ok {
		dst.CaseSensitive = &trueVal
	}

	return dst
}

func (ot *OptionsTemplate) MergeMword(dst *config.MwordOptions, src map[string]bool) *config.MwordOptions {
	trueVal, falseVal := true, false
	if dst == nil {
		if len(src) == 0 {
			return nil
		}
		dst = &config.MwordOptions{}
	}

	if _, ok := src["-always"]; ok {
		alwaysModeVal := config.AlwaysMode
		dst.Mode = &alwaysModeVal
	}

	if _, ok := src["-online"]; ok {
		onlineModeVal := config.OnlineMode
		dst.Mode = &onlineModeVal
	}

	if _, ok := src["-offline"]; ok {
		offlineModeVal := config.OfflineMode
		dst.Mode = &offlineModeVal
	}

	if _, ok := src["-nofirst"]; ok {
		dst.IsFirst = &falseVal
	}

	if _, ok := src["-first"]; ok {
		dst.IsFirst = &trueVal
	}

	if _, ok := src["-nosub"]; ok {
		dst.NoSub = &trueVal
	}

	if _, ok := src["-sub"]; ok {
		dst.NoSub = &falseVal
	}

	if _, ok := src["-novip"]; ok {
		dst.NoVip = &trueVal
	}

	if _, ok := src["-vip"]; ok {
		dst.NoVip = &falseVal
	}

	if _, ok := src["-norepeat"]; ok {
		dst.NoRepeat = &trueVal
	}

	if _, ok := src["-repeat"]; ok {
		dst.NoRepeat = &falseVal
	}

	if _, ok := src["-nooneword"]; ok {
		dst.OneWord = &falseVal
	}

	if _, ok := src["-oneword"]; ok {
		dst.OneWord = &trueVal
	}

	if _, ok := src["-nocontains"]; ok {
		dst.Contains = &falseVal
	}

	if _, ok := src["-contains"]; ok {
		dst.Contains = &trueVal
	}

	if _, ok := src["-nocase"]; ok {
		dst.CaseSensitive = &falseVal
	}

	if _, ok := src["-case"]; ok {
		dst.CaseSensitive = &trueVal
	}

	return dst
}

func (ot *OptionsTemplate) MergeTimer(dst *config.TimerOptions, src map[string]bool) *config.TimerOptions {
	if dst == nil {
		if len(src) == 0 {
			return nil
		}
		dst = &config.TimerOptions{}
	}

	if _, ok := src["-always"]; ok {
		alwaysModeVal := config.AlwaysMode
		dst.Mode = &alwaysModeVal
	}

	if _, ok := src["-online"]; ok {
		onlineModeVal := config.OnlineMode
		dst.Mode = &onlineModeVal
	}

	if _, ok := src["-offline"]; ok {
		offlineModeVal := config.OfflineMode
		dst.Mode = &offlineModeVal
	}

	if _, ok := src["-noa"]; ok {
		falseVal := false
		dst.IsAnnounce = &falseVal
	}

	if _, ok := src["-a"]; ok {
		trueVal := true
		dst.IsAnnounce = &trueVal
	}

	if _, ok := src["-primary"]; ok {
		primaryVal := "primary"
		dst.ColorAnnounce = &primaryVal
	}

	if _, ok := src["-blue"]; ok {
		blueVal := "blue"
		dst.ColorAnnounce = &blueVal
	}

	if _, ok := src["-green"]; ok {
		greenVal := "green"
		dst.ColorAnnounce = &greenVal
	}

	if _, ok := src["-orange"]; ok {
		orangeVal := "orange"
		dst.ColorAnnounce = &orangeVal
	}

	if _, ok := src["-purple"]; ok {
		purpleVal := "purple"
		dst.ColorAnnounce = &purpleVal
	}

	if dst.ColorAnnounce == nil || *dst.ColorAnnounce == "" {
		defaultVal := "primary"
		dst.ColorAnnounce = &defaultVal
	}

	return dst
}

func (ot *OptionsTemplate) MergeCommand(dst *config.CommandOptions, src map[string]bool) *config.CommandOptions {
	if dst == nil {
		if len(src) == 0 {
			return nil
		}
		dst = &config.CommandOptions{}
	}

	if _, ok := src["-public"]; ok {
		falseVal := false
		dst.IsPrivate = &falseVal
	}

	if _, ok := src["-private"]; ok {
		trueVal := true
		dst.IsPrivate = &trueVal
	}

	if _, ok := src["-always"]; ok {
		alwaysModeVal := config.AlwaysMode
		dst.Mode = &alwaysModeVal
	}

	if _, ok := src["-online"]; ok {
		onlineModeVal := config.OnlineMode
		dst.Mode = &onlineModeVal
	}

	if _, ok := src["-offline"]; ok {
		offlineModeVal := config.OfflineMode
		dst.Mode = &offlineModeVal
	}

	return dst
}

func (ot *OptionsTemplate) ExceptToString(opts *config.ExceptOptions) string {
	result := []string{
		func() string {
			if opts == nil || opts.NoRepeat == nil || !*opts.NoRepeat {
				return "-repeat"
			}
			return "-norepeat"
		}(),
		func() string {
			if opts == nil || opts.OneWord == nil || !*opts.OneWord {
				return "-nooneword"
			}
			return "-oneword"
		}(),
		func() string {
			if opts == nil || opts.Contains == nil || !*opts.Contains {
				return "-nocontains"
			}
			return "-contains"
		}(),
		func() string {
			if opts == nil || opts.CaseSensitive == nil || !*opts.CaseSensitive {
				return "-nocase"
			}
			return "-case"
		}(),
		func() string {
			if opts == nil || opts.NoSub == nil || *opts.NoSub {
				return "-nosub"
			}
			return "-sub"
		}(),
		func() string {
			if opts == nil || opts.NoVip == nil || *opts.NoVip {
				return "-novip"
			}
			return "-vip"
		}(),
	}
	return strings.Join(result, " ")
}

func (ot *OptionsTemplate) MwordToString(opts *config.MwordOptions) string {
	result := []string{
		func() string {
			if opts == nil || opts.Mode == nil {
				return "-online"
			}
			switch *opts.Mode {
			case config.OnlineMode:
				return "-online"
			case config.OfflineMode:
				return "-offline"
			default:
				return "-always"
			}
		}(),
		func() string {
			if opts == nil || opts.NoRepeat == nil || !*opts.NoRepeat {
				return "-repeat"
			}
			return "-norepeat"
		}(),
		func() string {
			if opts == nil || opts.OneWord == nil || !*opts.OneWord {
				return "-nooneword"
			}
			return "-oneword"
		}(),
		func() string {
			if opts == nil || opts.Contains == nil || !*opts.Contains {
				return "-nocontains"
			}
			return "-contains"
		}(),
		func() string {
			if opts == nil || opts.CaseSensitive == nil || !*opts.CaseSensitive {
				return "-nocase"
			}
			return "-case"
		}(),
		func() string {
			if opts == nil || opts.IsFirst == nil || !*opts.IsFirst {
				return "-nofirst"
			}
			return "-first"
		}(),
		func() string {
			if opts == nil || opts.NoSub == nil || *opts.NoSub {
				return "-nosub"
			}
			return "-sub"
		}(),
		func() string {
			if opts == nil || opts.NoVip == nil || *opts.NoVip {
				return "-novip"
			}
			return "-vip"
		}(),
	}
	return strings.Join(result, " ")
}

func (ot *OptionsTemplate) TimerToString(opts *config.TimerOptions) string {
	result := []string{
		func() string {
			if opts == nil || opts.IsAnnounce == nil || !*opts.IsAnnounce {
				return "-noa"
			}
			return "-a"
		}(),
		func() string {
			if opts == nil || opts.Mode == nil {
				return "-online"
			}
			switch *opts.Mode {
			case config.OnlineMode:
				return "-online"
			case config.OfflineMode:
				return "-offline"
			default:
				return "-always"
			}
		}(),
		func() string {
			if opts == nil || opts.ColorAnnounce == nil || *opts.ColorAnnounce == "" {
				return "-primary"
			}
			return "-" + *opts.ColorAnnounce
		}(),
	}
	return strings.Join(result, " ")
}

func (ot *OptionsTemplate) CommandToString(opts *config.CommandOptions) string {
	result := []string{
		func() string {
			if opts == nil || opts.IsPrivate == nil || !*opts.IsPrivate {
				return "-public"
			}
			return "-private"
		}(),
		func() string {
			if opts == nil || opts.Mode == nil {
				return "-always"
			}
			switch *opts.Mode {
			case config.OnlineMode:
				return "-online"
			case config.OfflineMode:
				return "-offline"
			default:
				return "-always"
			}
		}(),
	}
	return strings.Join(result, " ")
}
