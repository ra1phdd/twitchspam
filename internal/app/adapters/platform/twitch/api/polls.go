package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"twitchspam/internal/app/ports"
)

func (t *Twitch) CreatePoll(broadcasterID, title string, choices []string, duration int, enablePoints bool, pointsPerVote int) (*ports.Poll, error) {
	if broadcasterID == "" {
		return nil, errors.New("broadcasterID is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if len(choices) < 2 || len(choices) > 5 {
		return nil, errors.New("choices must contain between 2 and 5 elements")
	}
	if duration < 15 || duration > 1800 {
		return nil, errors.New("duration must be between 15 and 1800 seconds")
	}
	if enablePoints && (pointsPerVote < 1 || pointsPerVote > 1000000) {
		return nil, errors.New("channel_points_per_vote must be between 1 and 1000000")
	}

	choiceObjs := make([]PollChoice, len(choices))
	for i, c := range choices {
		choiceObjs[i] = PollChoice{Title: c}
	}

	opts := CreatePollOptions{
		BroadcasterID:              broadcasterID,
		Title:                      title,
		Choices:                    choiceObjs,
		Duration:                   duration,
		ChannelPointsVotingEnabled: enablePoints,
		ChannelPointsPerVote:       pointsPerVote,
	}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	token, err := t.ensureUserToken(context.Background(), broadcasterID)
	if err != nil {
		return nil, err
	}

	var createPollResp CreatePollResponse
	if statusCode, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/polls",
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, &createPollResp); err != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			return nil, ErrUserAuthNotCompleted
		case http.StatusBadRequest:
			return nil, ErrBadRequest
		case http.StatusTooManyRequests:
			return nil, ErrRateLimited
		default:
			return nil, err
		}
	}

	if len(createPollResp.Data) == 0 {
		return nil, errors.New("poll response is empty")
	}

	pollData := createPollResp.Data[0]
	startedAt, err := time.Parse(time.RFC3339, pollData.StartedAt)
	if err != nil {
		return nil, err
	}

	choicesResp := make([]ports.PollChoiceResponse, 0, len(pollData.Choices))
	for _, choice := range pollData.Choices {
		choicesResp = append(choicesResp, ports.PollChoiceResponse{
			ID:    choice.ID,
			Title: choice.Title,
		})
	}

	return &ports.Poll{
		ID:                         pollData.ID,
		Title:                      pollData.Title,
		Choices:                    choicesResp,
		Status:                     pollData.Status,
		Duration:                   pollData.Duration,
		StartedAt:                  startedAt.In(time.Local),
		ChannelPointsVotingEnabled: pollData.ChannelPointsVotingEnabled,
		ChannelPointsPerVote:       pollData.ChannelPointsPerVote,
	}, nil
}

func (t *Twitch) EndPoll(broadcasterID, pollID, status string) error {
	if broadcasterID == "" {
		return errors.New("broadcasterID is required")
	}
	if pollID == "" {
		return errors.New("poll id is required")
	}
	if status == "" {
		return errors.New("status is required")
	}

	validStatuses := map[string]struct{}{
		"TERMINATED": {},
		"ARCHIVED":   {},
	}

	if _, ok := validStatuses[status]; !ok {
		return errors.New("status must be one of: TERMINATED, ARCHIVED")
	}

	opts := EndPollOptions{
		BroadcasterID: broadcasterID,
		ID:            pollID,
		Status:        status,
	}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return err
	}

	token, err := t.ensureUserToken(context.Background(), broadcasterID)
	if err != nil {
		return err
	}

	if statusCode, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPatch,
		URL:    "https://api.twitch.tv/helix/polls",
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		switch statusCode {
		case http.StatusUnauthorized:
			return ErrUserAuthNotCompleted
		case http.StatusBadRequest:
			return ErrBadRequest
		case http.StatusNotFound:
			return ErrNotFound
		case http.StatusTooManyRequests:
			return ErrRateLimited
		default:
			return err
		}
	}

	return nil
}
