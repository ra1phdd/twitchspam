package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"twitchspam/internal/app/domain"
)

func (t *Twitch) SearchCategory(categoryName string) (string, string, error) {
	params := url.Values{}
	params.Set("query", categoryName)

	var searchResp SearchCategoriesResponse
	if _, err := t.doTwitchRequest(context.Background(), twitchRequest{
		Method: http.MethodGet,
		URL:    "https://api.twitch.tv/helix/search/categories?" + params.Encode(),
		Token:  nil,
		Body:   nil,
	}, &searchResp); err != nil {
		return "", "", err
	}

	if len(searchResp.Data) == 0 {
		return "", "", fmt.Errorf("game not found: %s", categoryName)
	}

	for _, g := range searchResp.Data {
		if strings.EqualFold(domain.RemovePunctuationOption.Fn(g.Name), domain.RemovePunctuationOption.Fn(categoryName)) {
			return g.ID, g.Name, nil
		}
	}

	return searchResp.Data[0].ID, searchResp.Data[0].Name, nil
}
