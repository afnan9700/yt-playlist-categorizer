package main

import (
	"context"
	"encoding/json"
	"flag"

	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	pb "github.com/afnan9700/yt-playlist-categorizer/proto"

	"google.golang.org/protobuf/encoding/protojson"
)

type fetchPlaylistReq struct {
	PlaylistID    string            `json:"playlistId"`
	Strategy      string            `json:"strategy,omitempty"`
	FetchChannels *bool             `json:"fetchChannels,omitempty"` // nil -> default true
	Params        map[string]string `json:"params,omitempty"`
}

func main() {
	addr := flag.String("addr", ":8080", "http listen address")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	apiKey := os.Getenv("YT_API_KEY")
	if apiKey == "" {
		log.Fatal("set YT_API_KEY environment variable")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/fetch-playlist", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		var reqBody fetchPlaylistReq
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if reqBody.PlaylistID == "" {
			http.Error(w, "playlistId required", http.StatusBadRequest)
			return
		}

		// default values
		strategy := "hdbscan"
		if reqBody.Strategy != "" {
			strategy = reqBody.Strategy
		}
		fetchChannels := true
		if reqBody.FetchChannels != nil {
			fetchChannels = *reqBody.FetchChannels
		}
		params := reqBody.Params
		if params == nil {
			params = map[string]string{
				"title_weight":   "0.8",
				"channel_weight": "0.2",
			}
		}

		// use a context with timeout for the external calls
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()

		items, err := fetchPlaylistItems(ctx, apiKey, reqBody.PlaylistID)
		if err != nil {
			http.Error(w, "failed fetch playlist: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// build unique uploader channel id set using VideoOwnerChannelId primarily
		channelSet := map[string]struct{}{}
		for _, it := range items {
			uid := it.Snippet.VideoOwnerChannelId
			if uid == "" {
				uid = it.Snippet.ChannelId
			}
			if uid != "" {
				channelSet[uid] = struct{}{}
			}
		}
		var channelIDs []string
		for k := range channelSet {
			channelIDs = append(channelIDs, k)
		}

		var channelDescMap map[string]string
		if fetchChannels && len(channelIDs) > 0 {
			m, err := fetchChannelDescriptions(ctx, apiKey, channelIDs)
			if err != nil {
				log.Printf("warning: failed to fetch channel descriptions: %v", err)
				channelDescMap = map[string]string{}
			} else {
				channelDescMap = m
			}
		} else {
			channelDescMap = map[string]string{}
		}

		// assemble protobuf ClusterRequest
		creq := &pb.ClusterRequest{
			Strategy: strategy,
			Params:   params,
		}

		for _, it := range items {
			videoID := it.ContentDetails.VideoId
			if videoID == "" {
				if v, ok := it.Snippet.ResourceId["videoId"]; ok {
					videoID = v
				}
			}
			// prefer uploader id fields
			uploaderID := it.Snippet.VideoOwnerChannelId
			if uploaderID == "" {
				uploaderID = it.Snippet.ChannelId
			}
			// pick channel description if available (uploader)
			chDesc := ""
			if uploaderID != "" {
				chDesc = channelDescMap[uploaderID]
			}

			v := &pb.Video{
				VideoId:            videoID,
				Title:              it.Snippet.Title,
				ChannelId:          uploaderID,
				ChannelDescription: chDesc,
				PlaylistId:         it.Snippet.PlaylistId,
				Position:           it.Snippet.Position,
			}
			creq.Videos = append(creq.Videos, v)
		}

		// return proto as JSON (protojson) so clients can see it or forward it to python
		marshaler := protojson.MarshalOptions{Multiline: true, Indent: "  "}
		js, err := marshaler.Marshal(creq)
		if err != nil {
			http.Error(w, "failed json marshal: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	})

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	log.Printf("listening %s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
