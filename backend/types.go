package main

type PlaylistItemsResponse struct {
	NextPageToken string             `json:"nextPageToken"`
	Items         []PlaylistItemWrap `json:"items"`
}

type PlaylistItemWrap struct {
	Snippet        PlaylistItemSnippet `json:"snippet"`
	ContentDetails struct {
		VideoId string `json:"videoId"`
	} `json:"contentDetails"`
}

type PlaylistItemSnippet struct {
	PublishedAt            string               `json:"publishedAt"`
	ChannelId              string               `json:"channelId"` // channel that added the item
	Title                  string               `json:"title"`
	Description            string               `json:"description"`
	Thumbnails             map[string]Thumbnail `json:"thumbnails"`
	ChannelTitle           string               `json:"channelTitle"`
	VideoOwnerChannelTitle string               `json:"videoOwnerChannelTitle"`
	VideoOwnerChannelId    string               `json:"videoOwnerChannelId"`
	PlaylistId             string               `json:"playlistId"`
	Position               int32                `json:"position"`
	ResourceId             map[string]string    `json:"resourceId"`
}

type Thumbnail struct {
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ChannelsResponse struct {
	Items []ChannelItem `json:"items"`
}

type ChannelItem struct {
	Id      string `json:"id"`
	Snippet struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"snippet"`
}
