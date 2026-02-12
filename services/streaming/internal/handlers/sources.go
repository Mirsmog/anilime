package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	streamingv1 "github.com/example/anime-platform/gen/streaming/v1"
	"github.com/example/anime-platform/internal/platform/api"
)

type sourceItem struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
	IsM3U8  bool   `json:"is_m3u8"`
}

type trackItem struct {
	Kind     string `json:"kind"`
	File     string `json:"file"`
	Label    string `json:"label"`
	Language string `json:"language"`
}

type introOutro struct {
	Start float32 `json:"start"`
	End   float32 `json:"end"`
}

type sourcesResponse struct {
	Sources           []sourceItem      `json:"sources"`
	Tracks            []trackItem       `json:"tracks"`
	Intro             *introOutro       `json:"intro,omitempty"`
	Outro             *introOutro       `json:"outro,omitempty"`
	Headers           map[string]string `json:"headers,omitempty"`
	ProviderEpisodeID string            `json:"provider_episode_id,omitempty"`
}

// Sources returns an HTTP handler that resolves streaming sources for an episode.
// It calls the streaming-resolver gRPC service and returns the playback data.
func Sources(client streamingv1.StreamingResolverServiceClient, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		episodeID := strings.TrimSpace(chi.URLParam(r, "anime_id"))
		if episodeID == "" {
			api.BadRequest(w, "MISSING_ID", "anime_id is required", "", nil)
			return
		}

		category := strings.TrimSpace(r.URL.Query().Get("category"))
		server := strings.TrimSpace(r.URL.Query().Get("server"))

		ctx := metadata.NewOutgoingContext(r.Context(), metadata.New(nil))
		resp, err := client.GetPlayback(ctx, &streamingv1.GetPlaybackRequest{
			EpisodeId: episodeID,
			Category:  category,
			Server:    server,
		})
		if err != nil {
			log.Warn("streaming-resolver GetPlayback failed", zap.String("episode_id", episodeID), zap.Error(err))
			api.WriteError(w, http.StatusBadGateway, "RESOLVER_UNAVAILABLE", "streaming resolver unavailable", "", nil)
			return
		}

		out := sourcesResponse{
			Sources: make([]sourceItem, 0, len(resp.GetSources())),
			Tracks:  make([]trackItem, 0, len(resp.GetTracks())),
		}
		for _, s := range resp.GetSources() {
			out.Sources = append(out.Sources, sourceItem{URL: s.GetUrl(), Quality: s.GetQuality(), IsM3U8: s.GetIsM3U8()})
		}
		for _, t := range resp.GetTracks() {
			out.Tracks = append(out.Tracks, trackItem{Kind: t.GetKind(), File: t.GetFile(), Label: t.GetLabel(), Language: t.GetLanguage()})
		}
		if resp.GetIntro() != nil && resp.GetIntro().GetEnd() > 0 {
			out.Intro = &introOutro{Start: resp.GetIntro().GetStart(), End: resp.GetIntro().GetEnd()}
		}
		if resp.GetOutro() != nil && resp.GetOutro().GetEnd() > 0 {
			out.Outro = &introOutro{Start: resp.GetOutro().GetStart(), End: resp.GetOutro().GetEnd()}
		}
		if resp.GetHeaders() != nil {
			out.Headers = resp.GetHeaders().GetHeaders()
		}
		out.ProviderEpisodeID = resp.GetProviderEpisodeId()

		api.WriteJSON(w, http.StatusOK, out)
	}
}
