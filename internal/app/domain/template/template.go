package template

import (
	"github.com/dlclark/regexp2"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type Template struct {
	aliases      *AliasesTemplate
	placeholders *PlaceholdersTemplate
	banwords     *BanwordsTemplate
	regex        *RegexTemplate
	options      *OptionsTemplate
}

func New(al map[string]string, banWords []string, banRegexps []*regexp2.Regexp, stream ports.StreamPort) *Template {
	return &Template{
		aliases:      NewAliases(al),
		placeholders: NewPlaceholders(stream),
		banwords:     NewBanwords(banWords, banRegexps),
		regex:        NewRegex(),
		options:      NewOptions(),
	}
}

func (t *Template) ReplaceAlias(text string) string {
	return t.aliases.replace(text)
}

func (t *Template) UpdateAliases(newAliases map[string]string) {
	t.aliases.update(newAliases)
}

func (t *Template) ReplacePlaceholders(text string, parts []string) string {
	return t.placeholders.replaceAll(text, parts)
}

func (t *Template) CheckOnBanwords(text, textOriginal string) bool {
	return t.banwords.checkMessage(text, textOriginal)
}

func (t *Template) MatchPhrase(words []string, phrase string) bool {
	return t.regex.matchPhrase(words, phrase)
}

func (t *Template) ParseOptions(words *[]string) (bool, config.Options) {
	return t.options.parse(words)
}

func (t *Template) MergeOptions(dst, src *config.Options) {
	t.options.merge(dst, src)
}
