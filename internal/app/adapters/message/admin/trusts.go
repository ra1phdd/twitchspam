package admin

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/trusts"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddRole struct {
	re     *regexp.Regexp
	trusts ports.TrustsPort
}

func (r *AddRole) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := r.re.FindStringSubmatch(msg.Message.Text.Text()) // !am role add <role_name> <scopes>
	if len(matches) != 3 {
		return nonParametr
	}

	name := strings.TrimSpace(matches[1])
	if _, ok := cfg.GlobalRoles[name]; ok {
		return existsRole
	}

	scopes := strings.Split(strings.TrimSpace(strings.ToLower(matches[2])), ",")
	found, notFound := make([]string, 0, len(scopes)), make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}

		if _, ok := trusts.ScopeMap[scope]; ok {
			found = append(found, scope)
			continue
		}
		notFound = append(notFound, scope)
	}

	if len(found) == 0 && len(notFound) == 0 {
		return nonParametr
	}
	current := cfg.Channels[channel].Roles[name]

	exists := make(map[string]struct{}, len(current))
	for _, s := range current {
		exists[s] = struct{}{}
	}

	for _, s := range found {
		if _, ok := exists[s]; !ok {
			current = append(current, s)
			exists[s] = struct{}{}
		}
	}

	cfg.Channels[channel].Roles[name] = current
	r.trusts.AddRole(name, found)

	return &ports.AnswerType{
		Text: []string{fmt.Sprintf("роль: %s, %s", name,
			buildResponse("скоупы не указаны", RespArg{Items: found, Name: "добавлены"}, RespArg{Items: notFound, Name: "не найдены"}).Text[0])},
		IsReply: true,
	}
}

type DelRole struct {
	re     *regexp.Regexp
	trusts ports.TrustsPort
}

func (r *DelRole) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := r.re.FindStringSubmatch(msg.Message.Text.Text()) // !am role del <role_name> <*scopes>
	if len(matches) != 3 {
		return nonParametr
	}

	name := strings.TrimSpace(matches[1])
	if _, ok := cfg.GlobalRoles[name]; ok {
		return &ports.AnswerType{
			Text:    []string{"данную роль нельзя удалить!"},
			IsReply: true,
		}
	}

	scopes := strings.Split(strings.TrimSpace(strings.ToLower(matches[2])), ",")
	if len(scopes) == 0 {
		delete(cfg.Channels[channel].Roles, name)
		r.trusts.DeleteRole(name, nil)
		return success
	}

	found, notFound := make([]string, 0, len(scopes)), make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}

		if _, ok := trusts.ScopeMap[scope]; ok {
			found = append(found, scope)
			continue
		}
		notFound = append(notFound, scope)
	}

	current := cfg.Channels[channel].Roles[name]
	toRemove := make(map[string]struct{}, len(found))
	for _, s := range found {
		toRemove[s] = struct{}{}
	}

	var updated []string
	for _, s := range current {
		if _, remove := toRemove[s]; !remove {
			updated = append(updated, s)
		}
	}

	if len(updated) == 0 {
		delete(cfg.Channels[channel].Roles, name)
		r.trusts.DeleteRole(name, nil)
		return success
	}

	cfg.Channels[channel].Roles[name] = updated
	r.trusts.DeleteRole(name, found)

	return &ports.AnswerType{
		Text: []string{fmt.Sprintf("роль: %s, %s", name,
			buildResponse("скоупы не указаны", RespArg{Items: found, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"}).Text[0])},
		IsReply: true,
	}
}

type TrustRole struct {
	re     *regexp.Regexp
	trusts ports.TrustsPort
	api    ports.APIPort
}

func (r *TrustRole) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := r.re.FindStringSubmatch(msg.Message.Text.Text()) // !am role trust <role_name> <users>
	if len(matches) != 3 {
		return nonParametr
	}

	name := strings.TrimSpace(matches[1])
	if _, ok := cfg.Channels[channel].Roles[name]; !ok {
		if _, ok := cfg.GlobalRoles[name]; !ok {
			return notFoundRole
		}
	}

	users := strings.Split(strings.ToLower(strings.TrimSpace(matches[2])), " ")
	ids, err := r.api.GetChannelIDs(users)
	if err != nil {
		return unknownError
	}

	found := make([]string, 0, len(ids))
	for user, id := range ids {
		user = strings.TrimSpace(user)
		if id == "" {
			continue
		}
		found = append(found, user)

		trust := cfg.Channels[channel].Trusts[id]
		if trust == nil {
			trust = &config.Trust{Username: user}
			cfg.Channels[channel].Trusts[id] = trust
		}

		if slices.Contains(trust.Roles, name) {
			continue
		}

		trust.Roles = append(trust.Roles, name)
		cfg.Channels[channel].Trusts[id].Roles = trust.Roles
		r.trusts.Update(id, trust.Roles, trust.Scopes)
	}

	return buildResponse("роли не указаны", RespArg{Items: found, Name: "добавлены"})
}

type UntrustRole struct {
	re     *regexp.Regexp
	trusts ports.TrustsPort
	api    ports.APIPort
}

func (r *UntrustRole) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := r.re.FindStringSubmatch(msg.Message.Text.Text()) // !am role untrust <role_name> <users>
	if len(matches) != 3 {
		return nonParametr
	}

	name := strings.TrimSpace(matches[1])
	if _, ok := cfg.Channels[channel].Roles[name]; !ok {
		if _, ok := cfg.GlobalRoles[name]; !ok {
			return notFoundRole
		}
	}

	users := strings.Split(strings.ToLower(strings.TrimSpace(matches[2])), " ")
	ids, err := r.api.GetChannelIDs(users)
	if err != nil {
		return unknownError
	}

	removed, notFound := make([]string, 0, len(ids)), make([]string, 0, len(ids))
	for user, id := range ids {
		user = strings.TrimSpace(user)
		if id == "" {
			continue
		}

		trust := cfg.Channels[channel].Trusts[id]
		if trust == nil {
			notFound = append(notFound, user)
			continue
		}

		newRoles := make([]string, 0, len(trust.Roles))
		for _, role := range trust.Roles {
			if role != name {
				newRoles = append(newRoles, role)
			}
		}

		if len(newRoles) != len(trust.Roles) {
			cfg.Channels[channel].Trusts[id].Roles = newRoles
			r.trusts.Update(id, newRoles, trust.Scopes)
			removed = append(removed, user)
		}

		if len(newRoles) == 0 && len(trust.Scopes) == 0 {
			delete(cfg.Channels[channel].Trusts, id)
			r.trusts.Update(id, nil, nil)
		}
	}

	return buildResponse("роли не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type ListRole struct {
	re *regexp.Regexp
	fs ports.FileServerPort
}

func (r *ListRole) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := r.re.FindStringSubmatch(msg.Message.Text.Text()) // !am role list <role_name?>
	if len(matches) != 2 {
		return nonParametr
	}

	type Role struct {
		Scopes []string
		Users  []string
	}

	roles := make(map[string]*Role)
	nameRole := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(matches[1]), "@"))
	for role, scopes := range cfg.Channels[channel].Roles {
		if strings.TrimSpace(nameRole) != "" && role != nameRole {
			continue
		}

		if _, ok := roles[role]; !ok {
			roles[role] = &Role{
				Scopes: scopes,
			}
		}

		for _, trust := range cfg.Channels[channel].Trusts {
			if !slices.Contains(trust.Roles, role) {
				continue
			}

			roles[role].Users = append(roles[role].Users, trust.Username)
		}
	}
	for role, scopes := range cfg.GlobalRoles {
		if strings.TrimSpace(nameRole) != "" && role != nameRole {
			continue
		}

		if _, ok := roles[role]; !ok {
			roles[role] = &Role{
				Scopes: scopes,
			}
		}

		for _, trust := range cfg.Channels[channel].Trusts {
			if !slices.Contains(trust.Roles, role) {
				continue
			}

			roles[role].Users = append(roles[role].Users, trust.Username)
		}
	}

	return buildList(roles, "роли", "роли не найдены!",
		func(name string, role *Role) string {
			var sb strings.Builder
			sb.WriteString(name + ":\n")
			sb.WriteString(fmt.Sprintf("- права доступа: %v\n", strings.Join(role.Scopes, ", ")))

			sb.WriteString("- пользователи с данной ролью: ")
			if len(role.Users) > 0 {
				sb.WriteString("\n")
				for _, user := range role.Users {
					sb.WriteString(fmt.Sprintf("  - %s\n", user))
				}
			} else {
				sb.WriteString("не найдены\n")
			}

			sb.WriteString("\n")
			return sb.String()
		}, r.fs)
}

type Trust struct {
	re     *regexp.Regexp
	trusts ports.TrustsPort
	api    ports.APIPort
}

func (t *Trust) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := t.re.FindStringSubmatch(msg.Message.Text.Text()) // !am trust <scope> <users>
	if len(matches) != 3 {
		return nonParametr
	}

	scope := strings.TrimSpace(strings.ToLower(matches[1]))
	users := strings.Split(strings.TrimSpace(strings.ToLower(matches[2])), " ")

	ids, err := t.api.GetChannelIDs(users)
	if err != nil {
		return unknownError
	}

	found := make([]string, 0, len(ids))
	for user, id := range ids {
		user = strings.TrimSpace(user)
		if id == "" {
			continue
		}
		found = append(found, user)

		trust := cfg.Channels[channel].Trusts[id]
		if trust == nil {
			trust = &config.Trust{Username: msg.Chatter.Login}
			cfg.Channels[channel].Trusts[id] = trust
		}

		if slices.Contains(trust.Scopes, scope) {
			continue
		}

		trust.Scopes = append(trust.Scopes, scope)
		cfg.Channels[channel].Trusts[id].Scopes = trust.Scopes
		t.trusts.Update(id, trust.Roles, trust.Scopes)
	}

	return buildResponse("пользователи не указаны", RespArg{Items: found, Name: "изменены"})
}

type Untrust struct {
	re     *regexp.Regexp
	trusts ports.TrustsPort
	api    ports.APIPort
}

func (u *Untrust) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := u.re.FindStringSubmatch(msg.Message.Text.Text()) // !am untrust <scope> <users>
	if len(matches) != 3 {
		return nonParametr
	}

	scope := strings.TrimSpace(strings.ToLower(matches[1]))
	users := strings.Split(strings.TrimSpace(strings.ToLower(matches[2])), " ")

	ids, err := u.api.GetChannelIDs(users)
	if err != nil {
		return unknownError
	}

	removed, notFound := make([]string, 0, len(users)), make([]string, 0, len(users))
	for user, id := range ids {
		user = strings.TrimSpace(user)
		if id == "" {
			continue
		}

		trust := cfg.Channels[channel].Trusts[id]
		if trust == nil {
			notFound = append(notFound, user)
			continue
		}

		newScopes := make([]string, 0, len(trust.Scopes))
		for _, sc := range trust.Scopes {
			if sc != scope {
				newScopes = append(newScopes, sc)
			}
		}

		if len(newScopes) != len(trust.Scopes) {
			cfg.Channels[channel].Trusts[id].Scopes = newScopes
			u.trusts.Update(id, trust.Roles, newScopes)
			removed = append(removed, user)
		}

		if len(newScopes) == 0 && len(trust.Roles) == 0 {
			delete(cfg.Channels[channel].Trusts, id)
			u.trusts.Update(id, nil, nil)
		}
	}

	return buildResponse("пользователи не указаны", RespArg{Items: removed, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type ListTrust struct {
	re *regexp.Regexp
	fs ports.FileServerPort
}

func (t *ListTrust) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := t.re.FindStringSubmatch(msg.Message.Text.Text()) // !am trust list <user?>
	if len(matches) != 2 {
		return nonParametr
	}

	user := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(matches[1]), "@"))
	return buildList(cfg.Channels[channel].Trusts, "доверенные пользователи", "доверенные пользователи не найдены!",
		func(name string, trust *config.Trust) string {
			if user != "" && user != trust.Username {
				return ""
			}

			if len(trust.Roles) == 0 {
				trust.Roles = append(trust.Roles, "не найдены")
			}

			if len(trust.Scopes) == 0 {
				trust.Scopes = append(trust.Scopes, "не найдены")
			}

			var sb strings.Builder
			sb.WriteString(trust.Username + ":\n")
			sb.WriteString(fmt.Sprintf("- роли: %v\n", strings.Join(trust.Roles, ", ")))
			sb.WriteString(fmt.Sprintf("- права доступа: %v\n", strings.Join(trust.Scopes, ", ")))
			sb.WriteString("\n")
			return sb.String()
		}, t.fs)
}
