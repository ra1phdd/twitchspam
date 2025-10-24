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

func (t *Twitch) CreatePrediction(broadcasterID, title string, outcomes []string, predictionWindow int) (*ports.Predictions, error) {
	if broadcasterID == "" {
		return nil, errors.New("broadcasterID is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if len(outcomes) < 2 {
		return nil, errors.New("at least 2 outcomes are required")
	}
	if predictionWindow < 30 || predictionWindow > 1800 {
		return nil, errors.New("prediction_window must be between 30 and 1800 seconds")
	}

	outcomeObjs := make([]Outcome, len(outcomes))
	for i, o := range outcomes {
		outcomeObjs[i] = Outcome{Title: o}
	}

	opts := CreatePredictionOptions{
		BroadcasterID:    broadcasterID,
		Title:            title,
		Outcomes:         outcomeObjs,
		PredictionWindow: predictionWindow,
	}

	bodyBytes, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	token, err := t.ensureUserToken(context.Background(), broadcasterID)
	if err != nil {
		return nil, err
	}

	var createPred CreatePredictionResponse
	if statusCode, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodPost,
		URL:    "https://api.twitch.tv/helix/predictions",
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, &createPred); err != nil {
		if statusCode == http.StatusUnauthorized {
			return nil, ErrUserAuthNotCompleted
		}
		if statusCode == http.StatusBadRequest {
			return nil, ErrBadRequest
		}
		if statusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		return nil, err
	}

	if len(createPred.Data) == 0 {
		return nil, errors.New("prediction response is empty")
	}

	createdAt, err := time.Parse(time.RFC3339, createPred.Data[0].CreatedAt)
	if err != nil {
		return nil, err
	}

	outcomesResp := make([]ports.PredictionsOutcome, 0, len(createPred.Data[0].Outcomes))
	for _, outcome := range createPred.Data[0].Outcomes {
		outcomesResp = append(outcomesResp, ports.PredictionsOutcome{
			ID:    outcome.ID,
			Title: outcome.Title,
		})
	}

	return &ports.Predictions{
		ID:               createPred.Data[0].ID,
		Title:            createPred.Data[0].Title,
		Outcomes:         outcomesResp,
		PredictionWindow: createPred.Data[0].PredictionWindow,
		Status:           createPred.Data[0].Status,
		CreatedAt:        createdAt.In(time.Local),
	}, nil
}

func (t *Twitch) EndPrediction(broadcasterID, predictionID, status, winningOutcomeID string) error {
	if broadcasterID == "" {
		return errors.New("broadcasterID is required")
	}
	if predictionID == "" {
		return errors.New("prediction id is required")
	}
	if status == "" {
		return errors.New("status is required")
	}

	validStatuses := map[string]struct{}{
		"RESOLVED": {},
		"CANCELED": {},
		"LOCKED":   {},
	}

	if _, ok := validStatuses[status]; !ok {
		return errors.New("status must be one of: RESOLVED, CANCELED, LOCKED")
	}

	if status == "RESOLVED" && winningOutcomeID == "" {
		return errors.New("winningOutcomeID is required when status is RESOLVED")
	}

	opts := EndPredictionOptions{
		BroadcasterID:    broadcasterID,
		ID:               predictionID,
		Status:           status,
		WinningOutcomeID: winningOutcomeID,
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
		URL:    "https://api.twitch.tv/helix/predictions",
		Token:  token,
		Body:   bytes.NewReader(bodyBytes),
	}, nil); err != nil {
		if statusCode == http.StatusUnauthorized {
			return ErrUserAuthNotCompleted
		}
		if statusCode == http.StatusBadRequest {
			return ErrBadRequest
		}
		if statusCode == http.StatusNotFound {
			return ErrNotFound
		}
		if statusCode == http.StatusTooManyRequests {
			return ErrRateLimited
		}
		return err
	}

	return nil
}
