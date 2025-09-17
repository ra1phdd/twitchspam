package template

import (
	"github.com/dlclark/regexp2"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type MwordTemplate struct {
	log logger.Logger

	trie ports.TriePort[mwordMeta]
	re   *regexp2.Regexp

	mwords  map[string]mwordMeta
	regexps map[*regexp2.Regexp]mwordMeta
}

type mwordMeta struct {
	punishments *[]config.Punishment
	options     *config.SpamOptions
}

func NewMword(log logger.Logger, mwordGroups map[string]*config.MwordGroup, mwords map[string]*config.Mword) *MwordTemplate {
	mt := &MwordTemplate{
		log:     log,
		mwords:  make(map[string]mwordMeta),
		regexps: make(map[*regexp2.Regexp]mwordMeta),
	}
	mt.update(mwordGroups, mwords)

	return mt
}

func (mt *MwordTemplate) update(mwordGroups map[string]*config.MwordGroup, mwords map[string]*config.Mword) {
	var patterns []string
	m := make(map[string]mwordMeta)
	for _, group := range mwordGroups {
		if !group.Enabled {
			continue
		}

		for _, w := range group.Words {
			w = strings.TrimSpace(w)
			if w == "" {
				continue
			}

			meta := mwordMeta{
				punishments: &group.Punishments,
				options:     group.Options,
			}

			m[w] = meta
			mt.mwords[w] = meta
		}

		for _, re := range group.Regexp {
			patterns = append(patterns, re.String())

			meta := mwordMeta{
				punishments: &group.Punishments,
				options:     group.Options,
			}
			mt.regexps[re] = meta
		}
	}

	for w, mword := range mwords {
		if mword.Regexp != nil {
			patterns = append(patterns, mword.Regexp.String())

			meta := mwordMeta{
				punishments: &mword.Punishments,
				options:     mword.Options,
			}
			mt.regexps[mword.Regexp] = meta

			continue
		}

		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		meta := mwordMeta{
			punishments: &mword.Punishments,
			options:     mword.Options,
		}

		m[w] = meta
		mt.mwords[w] = meta
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

func (mt *MwordTemplate) match(text string, words []string) (bool, []config.Punishment, config.SpamOptions) {
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
				return true, *curValue.punishments, *curValue.options
			}
		}
	}

	if mt.re != nil {
		if isMatch, _ := mt.re.MatchString(text); !isMatch {
			return false, nil, config.SpamOptions{}
		}

		for re, meta := range mt.regexps {
			if isMatch, _ := re.MatchString(text); isMatch {
				return true, *meta.punishments, *meta.options
			}
		}
	}

	return false, nil, config.SpamOptions{}
}
