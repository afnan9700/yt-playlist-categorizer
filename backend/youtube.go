package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// fetchPlaylistItems fetches all items for a playlist (handles pagination).
// Requires a valid API key. Returns the raw PlaylistItemWrap list.
func fetchPlaylistItems(ctx context.Context, apiKey, playlistID string) ([]PlaylistItemWrap, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	base := "https://www.googleapis.com/youtube/v3/playlistItems"
	var all []PlaylistItemWrap
	pageToken := ""
	for {
		u, _ := url.Parse(base)
		q := u.Query()
		q.Set("part", "snippet,contentDetails")
		q.Set("playlistId", playlistID)
		q.Set("maxResults", "50")
		q.Set("key", apiKey)
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		u.RawQuery = q.Encode()
		req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("youtube request failed: %w", err)
		}
		if resp.StatusCode != 200 {
			var b struct{ Error interface{} }
			_ = json.NewDecoder(resp.Body).Decode(&b)
			resp.Body.Close()
			return nil, fmt.Errorf("youtube api non-200: %d: %#v", resp.StatusCode, b)
		}
		var page PlaylistItemsResponse
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode playlist page: %w", err)
		}
		resp.Body.Close()
		for _, it := range page.Items {
			all = append(all, it)
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return all, nil
}

// fetchChannelDescriptions collects unique channel IDs and fetches their snippet descriptions.
// channels.list supports up to 50 ids per call.
func fetchChannelDescriptions(ctx context.Context, apiKey string, channelIDs []string) (map[string]string, error) {
	out := map[string]string{}
	client := &http.Client{Timeout: 15 * time.Second}
	base := "https://www.googleapis.com/youtube/v3/channels"

	batch := func(ids []string) error {
		u, _ := url.Parse(base)
		q := u.Query()
		q.Set("part", "snippet")
		q.Set("id", strings.Join(ids, ","))
		q.Set("key", apiKey)
		u.RawQuery = q.Encode()
		req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			var b struct{ Error interface{} }
			_ = json.NewDecoder(resp.Body).Decode(&b)
			resp.Body.Close()
			return fmt.Errorf("channel list non-200: %d %#v", resp.StatusCode, b)
		}
		var cr ChannelsResponse
		if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
			resp.Body.Close()
			return err
		}
		resp.Body.Close()
		for _, item := range cr.Items {
			out[item.Id] = item.Snippet.Description
		}
		return nil
	}

	// chunk into batches of up to 50
	for i := 0; i < len(channelIDs); i += 50 {
		j := i + 50
		if j > len(channelIDs) {
			j = len(channelIDs)
		}
		if err := batch(channelIDs[i:j]); err != nil {
			return nil, err
		}
	}
	return out, nil
}
