package handlers

import (
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	streamingv1 "github.com/example/anime-platform/gen/streaming/v1"
	"github.com/example/anime-platform/internal/platform/analytics"
	"github.com/example/anime-platform/internal/platform/api"
	"github.com/example/anime-platform/internal/platform/auth"
	"github.com/example/anime-platform/internal/platform/httpserver"
	"github.com/example/anime-platform/internal/platform/signing"
)

type watchResponse struct {
	Sources           []*streamingv1.PlaybackSource   `json:"sources"`
	Tracks            []*streamingv1.PlaybackTrack    `json:"tracks"`
	Intro             *streamingv1.PlaybackIntroOutro `json:"intro,omitempty"`
	Outro             *streamingv1.PlaybackIntroOutro `json:"outro,omitempty"`
	SignedPlaybackURL string                          `json:"signed_playback_url"`
}

func Watch(client streamingv1.StreamingResolverServiceClient, hlsBase, hlsSecret string, ap *analytics.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := httpserver.RequestIDFromContext(r.Context())
		uid, ok := auth.UserIDFromContext(r.Context())
		if !ok || strings.TrimSpace(uid) == "" {
			api.Unauthorized(w, "AUTH_MISSING", "Missing auth", rid)
			return
		}

		episodeID := strings.TrimSpace(r.PathValue("episode_id"))
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		server := strings.TrimSpace(r.URL.Query().Get("server"))

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := client.GetPlayback(ctx, &streamingv1.GetPlaybackRequest{EpisodeId: episodeID, Category: category, Server: server})
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Message() != "" {
				api.WriteError(w, http.StatusBadGateway, "STREAMING_UNAVAILABLE", st.Message(), rid, nil)
				return
			}
			writeGRPCError(w, rid, err)
			return
		}

		if len(resp.GetSources()) == 0 {
			api.NotFound(w, "NO_SOURCES", "No playback sources", rid)
			return
		}

		primary := resp.GetSources()[0].GetUrl()
		s := signing.New(hlsSecret)

		// Get headers from streaming response
		var hdrs map[string]string
		if h := resp.GetHeaders(); h != nil {
			hdrs = h.GetHeaders()
		}

		signed := s.SignWithHeaders(primary, uid, time.Now().Add(15*time.Minute), hdrs)
		url, err := signing.BuildSignedURL(hlsBase+"/hls", signed)
		if err != nil {
			api.Internal(w, rid)
			return
		}

		ap.Publish(analytics.SubjectStreamingStarted, "playback_started", uid, map[string]any{
			"episode_id": episodeID,
			"category":   category,
		})
		api.WriteJSON(w, http.StatusOK, watchResponse{Sources: resp.GetSources(), Tracks: resp.GetTracks(), Intro: resp.GetIntro(), Outro: resp.GetOutro(), SignedPlaybackURL: url})
	}
}
