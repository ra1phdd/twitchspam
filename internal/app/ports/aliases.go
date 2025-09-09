package ports

type AliasesPort interface {
	Update(newAliases map[string]string)
	ReplaceOne(text string) string
	ReplacePlaceholders(text string, parts []string) string
}
