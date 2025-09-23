package template

import (
	"github.com/dlclark/regexp2"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type ExceptTemplate struct {
	log logger.Logger

	trie ports.TriePort[exceptMeta]
	re   *regexp2.Regexp

	except  map[string]exceptMeta
	regexps map[*regexp2.Regexp]exceptMeta
}

type exceptMeta struct {
	messageLimit int
	punishments  *[]config.Punishment
	options      *config.SpamOptions
}

func NewExcept(log logger.Logger, except map[string]*config.ExceptionsSettings) *ExceptTemplate {
	et := &ExceptTemplate{
		log:     log,
		except:  make(map[string]exceptMeta),
		regexps: make(map[*regexp2.Regexp]exceptMeta),
	}
	et.update(except)

	return et
}

func (mt *ExceptTemplate) update(except map[string]*config.ExceptionsSettings) {
	var patterns []string
	m := make(map[string]exceptMeta)
	for w, ex := range except {
		if !ex.Enabled {
			continue
		}

		if ex.Regexp != nil {
			patterns = append(patterns, ex.Regexp.String())

			meta := exceptMeta{
				messageLimit: ex.MessageLimit,
				punishments:  &ex.Punishments,
				options:      ex.Options,
			}
			mt.regexps[ex.Regexp] = meta
			continue
		}

		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		meta := exceptMeta{
			messageLimit: ex.MessageLimit,
			punishments:  &ex.Punishments,
			options:      ex.Options,
		}

		m[w] = meta
		mt.except[w] = meta
	}

	mt.trie = trie.NewTrie(m)
	if len(patterns) > 0 {
		var err error
		combinedPattern := "(?i)(" + strings.Join(patterns, "|") + ")"
		mt.re, err = regexp2.Compile(combinedPattern, regexp2.IgnoreCase)
		if err != nil {
			mt.log.Error("Failed to compile regexp on banwords", err)
		}
	}
}

func (mt *ExceptTemplate) match(text string, words []string) (bool, int, []config.Punishment, config.SpamOptions) {
	for i := 0; i < len(words); i++ {
		cur := mt.trie.Root()
		j := i
		for j < len(words) {
			next, ok := cur.Children()[words[j]]
			if !ok {
				break
			}
			cur = next
			j++
			if curValue := cur.Value(); curValue != nil {
				var opts config.SpamOptions
				if curValue.options != nil {
					opts = *curValue.options
				}
				var punish []config.Punishment
				if curValue.punishments != nil {
					punish = *curValue.punishments
				}
				return true, curValue.messageLimit, punish, opts
			}
		}
	}

	if mt.re != nil {
		if isMatch, _ := mt.re.MatchString(text); !isMatch {
			return false, 0, nil, config.SpamOptions{}
		}

		for re, meta := range mt.regexps {
			if isMatch, _ := re.MatchString(text); isMatch {
				var opts config.SpamOptions
				if meta.options != nil {
					opts = *meta.options
				}
				var punish []config.Punishment
				if meta.punishments != nil {
					punish = *meta.punishments
				}
				return true, meta.messageLimit, punish, opts
			}
		}
	}

	return false, 0, nil, config.SpamOptions{}
}
