package ports

import "twitchspam/internal/app/infrastructure/config"

type TemplatePort interface {
	ReplaceAlias(text string) string
	UpdateAliases(newAliases map[string]string)
	ReplacePlaceholders(text string, parts []string) string
	CheckOnBanwords(text string, wordsOriginal []string) bool
	MatchPhrase(words []string, phrase string) bool
	ParseOptions(words *[]string, opts map[string]struct{}) map[string]bool
	ParseOption(words *[]string, opt string) *bool
	ParseIntArg(valStr string, min, max int) (int, bool)
	ParseFloatArg(valStr string, min, max float64) (float64, bool)
	ParsePunishment(punishment string, allowInherit bool) (config.Punishment, error)
	FormatPunishments(punishments []config.Punishment) []string
	FormatPunishment(punishment config.Punishment) string
	UpdateMwords(mwordGroups map[string]*config.MwordGroup, mwords map[string]*config.Mword)
	MatchMwords(text string, words []string) (bool, []config.Punishment, config.SpamOptions)
	UpdateExcept(exDefault map[string]*config.ExceptionsSettings)
	MatchExcept(text string, words []string, countSpam int) (bool, []config.Punishment, config.SpamOptions)
	UpdateExceptEmote(exEmote map[string]*config.ExceptionsSettings)
	MatchExceptEmote(text string, words []string, countSpam int) (bool, []config.Punishment, config.SpamOptions)
}
