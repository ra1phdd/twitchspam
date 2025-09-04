package admin

import (
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleDelayAutomod(cfg *config.Config, _ string, args []string) ports.ActionType {
	if val, ok := parseIntArg(args, 0, 10); ok {
		cfg.Spam.DelayAutomod = val
		return None
	}
	return ErrDelayAutomod
}
