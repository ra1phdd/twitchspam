package ports

import (
	"regexp"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
)

type TemplatePort interface {
	CleanMessage(text string) string
	Aliases() AliasesPort
	Placeholders() PlaceholdersPort
	Banwords() BanwordsPort
	Options() OptionsPort
	Parser() ParserPort
	Punishment() PunishmentPort
	SpamPause() SpamPausePort
	Mword() MwordPort
	Store() StoresPort
	Nuke() NukePort
}

type AliasesPort interface {
	Update(newAliases map[string]string, newAliasGroups map[string]*config.AliasGroups, globalAliases map[string]string)
	Replace(parts []string) (string, bool)
}

type PlaceholdersPort interface {
	ReplaceAll(text string, parts []string) string
}

type BanwordsPort interface {
	CheckMessage(textLower string, wordsOriginal []string) bool
}

type OptionsPort interface {
	ParseAll(input string, opts map[string]struct{}) (string, map[string]bool)
	MergeExcept(dst config.ExceptOptions, src map[string]bool, isDefault bool) config.ExceptOptions
	MergeEmoteExcept(dst config.ExceptOptions, src map[string]bool, isDefault bool) config.ExceptOptions
	MergeMword(dst config.MwordOptions, src map[string]bool) config.MwordOptions
	MergeTimer(dst config.TimerOptions, src map[string]bool) config.TimerOptions
	MergeCommand(dst config.CommandOptions, src map[string]bool) config.CommandOptions
	ExceptToString(opts config.ExceptOptions) string
	MwordToString(opts config.MwordOptions) string
	TimerToString(opts config.TimerOptions) string
	CommandToString(opts config.CommandOptions) string
}

type ParserPort interface {
	ParseIntArg(valStr string, min int, max int) (int, bool)
	ParseFloatArg(valStr string, min float64, max float64) (float64, bool)
}

type PunishmentPort interface {
	Parse(punishment string, allowInherit bool) (config.Punishment, error)
	Get(arr []config.Punishment, idx int) (string, time.Duration)
	FormatAll(punishments []config.Punishment) []string
	Format(punishment config.Punishment) string
}

type SpamPausePort interface {
	Pause(duration time.Duration)
	CanProcess() bool
}

type MwordPort interface {
	Update(mwords []config.Mword, mwordGroups map[string]*config.MwordGroup)
	Check(msg *domain.ChatMessage) []config.Punishment
}

type StoresPort interface {
	SetMessageCapacity(cfg *config.Config)
	Messages() StorePort[storage.Message]
	Timeouts() StorePort[int]
}

type NukePort interface {
	Start(punishment config.Punishment, containsWords, words []string, regexp *regexp.Regexp)
	Cancel()
	Check(text *domain.MessageText) *CheckerAction
}
