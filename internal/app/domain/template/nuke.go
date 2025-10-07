package template

import (
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type NukeTemplate struct {
	nuke  *Nuke
	timer *time.Timer
	mu    sync.Mutex
}

type Nuke struct {
	expiresAt     time.Time
	punishment    config.Punishment
	containsWords []string
	words         []string
	regexp        *regexp.Regexp
}

func NewNuke() *NukeTemplate {
	return &NukeTemplate{}
}

func (n *NukeTemplate) Start(punishment config.Punishment, containsWords, words []string, regexp *regexp.Regexp) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}

	n.nuke = &Nuke{
		expiresAt:     time.Now().Add(5 * time.Minute),
		punishment:    punishment,
		containsWords: containsWords,
		words:         words,
		regexp:        regexp,
	}

	n.timer = time.AfterFunc(5*time.Minute, func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.nuke = nil
		n.timer = nil
	})
}

func (n *NukeTemplate) Cancel() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}
	n.nuke = nil
}

func (n *NukeTemplate) Check(text *domain.MessageText) *ports.CheckerAction {
	if n.nuke == nil {
		return nil
	}

	apply := func() *ports.CheckerAction {
		return &ports.CheckerAction{
			Type:     n.nuke.punishment.Action,
			Reason:   "массбан",
			Duration: time.Duration(n.nuke.punishment.Duration) * time.Second,
		}
	}

	if n.nuke.regexp != nil && n.nuke.regexp.MatchString(text.Text(domain.RemoveDuplicateLetters)) {
		return apply()
	}

	for _, w := range n.nuke.containsWords {
		if strings.Contains(text.Text(domain.RemoveDuplicateLetters), w) {
			return apply()
		}
	}

	for _, w := range n.nuke.words {
		if slices.Contains(text.Words(domain.RemoveDuplicateLetters), w) {
			return apply()
		}
	}

	return nil
}
