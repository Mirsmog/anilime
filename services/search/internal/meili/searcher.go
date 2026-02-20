package meili

import "context"

// Searcher is the port for querying the search backend.
type Searcher interface {
	Search(ctx context.Context, index string, payload any) (SearchResponse, error)
}
