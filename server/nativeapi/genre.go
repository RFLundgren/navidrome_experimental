package nativeapi

import (
	"net/http"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
)

func (api *Router) addGenreRandomSongsRoute(r chi.Router) {
	r.Get("/genre/{id}/randomSongs", api.genreRandomSongs())
}

// genreIDFilter mirrors persistence's unexported tagIDFilter for the "genre" tag - matching by
// the tag's ID (not name), consistent with the genre_id filter used everywhere else in the app.
// "tags" is a plain media_file column (scanner-populated JSON), so this needs no join.
func genreIDFilter(genreID string) Sqlizer {
	return persistence.Exists(`json_tree(tags, "$.genre")`, And{
		NotEq{"json_tree.atom": nil},
		Eq{"value": genreID},
	})
}

// genreRandomSongs returns up to `count` random song IDs from the given genre, filtered per
// randomPlaylistOptions, for the "Create Playlist from Genre" web UI action. GetRandom's first
// pass queries media_file alone (no annotation join), so "excludeSkipped" is applied in Go after
// hydration, using the Skipped field GetRandom's second pass does populate, rather than as a
// GetRandom-level SQL filter.
func (api *Router) genreRandomSongs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		genreID := chi.URLParam(r, "id")
		if genreID == "" {
			http.Error(w, "genre id is required", http.StatusBadRequest)
			return
		}

		opts := parseRandomPlaylistOptions(r)

		ctx := r.Context()
		candidates, err := api.ds.MediaFile(ctx).GetRandom(model.QueryOptions{
			Filters: And{genreIDFilter(genreID), Eq{"missing": false}},
			Max:     opts.count * genrePlaylistOverfetchFactor,
		})
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		_ = rest.RespondWithJSON(w, http.StatusOK, buildRandomPlaylist(candidates, opts))
	}
}
