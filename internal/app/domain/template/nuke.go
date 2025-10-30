package template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type NukeTemplate struct {
	nuke    *Nuke
	oldNuke *Nuke
	timer   *time.Timer
	mu      sync.Mutex
}

type Nuke struct {
	expiresAt     time.Time
	duration      time.Duration
	punishment    config.Punishment
	containsWords []string
	words         []string
	regexp        *regexp.Regexp

	cancel  context.CancelFunc
	startFn func(ctx context.Context)
}

func NewNuke() *NukeTemplate {
	return &NukeTemplate{}
}

func (n *NukeTemplate) Start(punishment config.Punishment, duration time.Duration, containsWords, words []string, regexp *regexp.Regexp, startFn func(ctx context.Context)) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}

	if n.nuke != nil && n.nuke.cancel != nil {
		n.nuke.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.nuke = &Nuke{
		expiresAt:     time.Now().Add(duration),
		duration:      duration,
		punishment:    punishment,
		containsWords: containsWords,
		words:         words,
		regexp:        regexp,
		cancel:        cancel,
		startFn:       startFn,
	}

	n.timer = time.AfterFunc(duration, func() {
		n.mu.Lock()
		defer n.mu.Unlock()

		n.oldNuke = n.nuke
		n.nuke = nil
		n.timer = nil
	})

	if startFn != nil {
		go startFn(ctx)
	}
}

func (n *NukeTemplate) Restart() error {
	if n.oldNuke == nil {
		return errors.New("a repeat of the previous nuke is not possible")
	}

	n.Start(n.oldNuke.punishment, n.oldNuke.duration, n.oldNuke.containsWords, n.oldNuke.words, n.oldNuke.regexp, n.oldNuke.startFn)
	return nil
}

func (n *NukeTemplate) Cancel() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.nuke == nil {
		return
	}

	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}

	if n.nuke.cancel != nil {
		n.nuke.cancel()
	}

	n.oldNuke = n.nuke
	n.nuke = nil
}

func (n *NukeTemplate) Check(text *message.Text, ignoreNuke bool) *ports.CheckerAction {
	if n.nuke == nil || ignoreNuke {
		return nil
	}

	apply := func() *ports.CheckerAction {
		return &ports.CheckerAction{
			Type:       n.nuke.punishment.Action,
			ReasonMod:  "массбан",
			ReasonUser: fmt.Sprintf("Не используй запрещенное слово!"),
			Duration:   time.Duration(n.nuke.punishment.Duration) * time.Second,
		}
	}

	if n.nuke.regexp != nil && n.nuke.regexp.MatchString(text.Text()) {
		return apply()
	}

	for _, w := range n.nuke.containsWords {
		if strings.Contains(
			text.Text(message.LowerOption, message.RemoveDuplicateLettersOption),
			(&message.Text{Original: w}).Text(message.LowerOption, message.RemoveDuplicateLettersOption),
		) {
			return apply()
		}
	}

	for _, w := range n.nuke.words {
		if strings.Contains(
			text.Text(message.LowerOption, message.RemoveDuplicateLettersOption),
			(&message.Text{Original: w}).Text(message.LowerOption, message.RemoveDuplicateLettersOption),
		) {
			return apply()
		}
	}

	return nil
}
