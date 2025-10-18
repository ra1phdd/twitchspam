package api

import (
	"fmt"
	"net/url"
	"strings"
	"twitchspam/internal/app/domain"
)

func (t *Twitch) SearchCategory(categoryName string) (string, string, error) {
	params := url.Values{}
	params.Set("query", categoryName)

	var searchResp SearchCategoriesResponse
	err := t.doTwitchRequest("GET", "https://api.twitch.tv/helix/search/categories?"+params.Encode(), nil, nil, &searchResp)
	if err != nil {
		t.log.Error("Failed to twitch request", err)
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
