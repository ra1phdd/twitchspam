package ports

import "twitchspam/internal/app/domain/trusts"

type TrustsPort interface {
	Update(user string, roles, scopes []string)
	AddRole(role string, scopes []string)
	DeleteRole(role string, scopes []string)
	HasScope(user string, scope trusts.Scope) bool
	HasAnyScope(user string, scopes ...trusts.Scope) bool
	GetScopes(user string) []string
}
