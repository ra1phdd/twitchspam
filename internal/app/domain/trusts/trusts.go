package trusts

import (
	"sync"
	"twitchspam/internal/app/infrastructure/config"
)

type Scope uint64

const (
	ScopeIgnoreAntispam Scope = 1 << iota
	ScopeIgnoreMword
	ScopeIgnoreBanwords
	ScopeIgnoreAds
	ScopeModActions
	ScopeNuke
	ScopePolls
	ScopePredictions
)

var ScopeMap = map[string]Scope{
	"noas": ScopeIgnoreAntispam,
	"nomw": ScopeIgnoreMword,
	"nobw": ScopeIgnoreBanwords,
	"noad": ScopeIgnoreAds,
	"mod":  ScopeModActions,
	"nuke": ScopeNuke,
	"poll": ScopePolls,
	"pred": ScopePredictions,
}

type TrustManager struct {
	roles       map[string][]string
	globalRoles map[string][]string
	users       map[string]*config.Trust

	trusts    sync.Map
	roleMasks sync.Map
}

func New(roles, globalRoles map[string][]string, users map[string]*config.Trust) *TrustManager {
	m := &TrustManager{
		roles:       roles,
		globalRoles: globalRoles,
		users:       users,
	}

	for role, scopes := range globalRoles {
		var mask Scope
		for _, s := range scopes {
			if val, ok := ScopeMap[s]; ok {
				mask |= val
			}
		}
		m.roleMasks.Store(role, mask)
	}

	for role, scopes := range roles {
		var mask Scope
		for _, s := range scopes {
			if val, ok := ScopeMap[s]; ok {
				mask |= val
			}
		}

		if val, ok := m.roleMasks.Load(role); ok {
			mask |= val.(Scope)
		}
		m.roleMasks.Store(role, mask)
	}

	for user, info := range users {
		mask := m.calcUserMask(info.Roles, info.Scopes)
		m.trusts.Store(user, mask)
	}

	return m
}

func (m *TrustManager) Update(user string, roles, scopes []string) {
	if roles == nil && scopes == nil {
		m.trusts.Delete(user)
		return
	}

	mask := m.calcUserMask(roles, scopes)
	m.trusts.Store(user, mask)
}

func (m *TrustManager) AddRole(role string, scopes []string) {
	currentScopes := m.roles[role]

	existing := make(map[string]struct{}, len(currentScopes))
	for _, s := range currentScopes {
		existing[s] = struct{}{}
	}

	for _, s := range scopes {
		if _, exists := existing[s]; !exists {
			currentScopes = append(currentScopes, s)
			existing[s] = struct{}{}
		}
	}
	m.roles[role] = currentScopes

	var mask Scope
	for _, s := range currentScopes {
		if val, ok := ScopeMap[s]; ok {
			mask |= val
		}
	}
	m.roleMasks.Store(role, mask)

	for user, info := range m.users {
		newMask := m.calcUserMask(info.Roles, info.Scopes)
		m.trusts.Store(user, newMask)
	}
}

func (m *TrustManager) DeleteRole(role string, scopes []string) {
	currentScopes, ok := m.roles[role]
	if !ok {
		return
	}

	if len(scopes) > 0 {
		scopeSet := make(map[string]struct{}, len(scopes))
		for _, s := range scopes {
			scopeSet[s] = struct{}{}
		}

		var updated []string
		for _, s := range currentScopes {
			if _, remove := scopeSet[s]; !remove {
				updated = append(updated, s)
			}
		}

		if len(updated) == 0 {
			delete(m.roles, role)
			m.roleMasks.Delete(role)
		} else {
			m.roles[role] = updated

			var mask Scope
			for _, s := range updated {
				if val, ok := ScopeMap[s]; ok {
					mask |= val
				}
			}
			m.roleMasks.Store(role, mask)
		}
	} else {
		delete(m.roles, role)
		m.roleMasks.Delete(role)
	}

	for user, info := range m.users {
		newMask := m.calcUserMask(info.Roles, info.Scopes)
		m.trusts.Store(user, newMask)
	}
}

func (m *TrustManager) HasScope(user string, scope Scope) bool {
	val, ok := m.trusts.Load(user)
	if !ok {
		return false
	}
	return val.(Scope)&scope != 0
}

func (m *TrustManager) HasAnyScope(user string, scopes ...Scope) bool {
	val, ok := m.trusts.Load(user)
	if !ok {
		return false
	}
	userMask := val.(Scope)

	var combined Scope
	for _, s := range scopes {
		combined |= s
	}
	return userMask&combined != 0
}

func (m *TrustManager) GetScopes(user string) []string {
	val, ok := m.trusts.Load(user)
	if !ok {
		return nil
	}
	mask := val.(Scope)
	var scopes []string
	for name, bit := range ScopeMap {
		if mask&bit != 0 {
			scopes = append(scopes, name)
		}
	}
	return scopes
}

func (m *TrustManager) calcUserMask(roles, scopes []string) Scope {
	var mask Scope
	for _, r := range roles {
		if val, ok := m.roleMasks.Load(r); ok {
			mask |= val.(Scope)
		}
	}

	for _, s := range scopes {
		if val, ok := ScopeMap[s]; ok {
			mask |= val
		}
	}

	return mask
}
