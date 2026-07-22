package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

func (api *Router) addMediaFileTagRoutes(r chi.Router) {
	r.Route("/mediaFileTag", func(r chi.Router) {
		r.Get("/", api.tagsForSong())
		r.Get("/names", api.allTagNames())
		r.Get("/counts", api.tagCounts())
		r.Get("/{tag}/randomSongs", api.tagRandomSongs())
		r.Post("/", api.tagSong())
		r.Delete("/", api.untagSong())
	})
}

type mediaFileTagPayload struct {
	MediaFileID string `json:"mediaFileId"`
	TagName     string `json:"tagName"`
}

// tagsForSong and allTagNames both accept an optional ?source=ai|user query
// param to narrow the result to one source; omitting it returns tags of any
// source, preserving this endpoint's original (pre-source) behavior.
func (api *Router) tagsForSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mediaFileID := r.URL.Query().Get("media_file_id")
		if mediaFileID == "" {
			http.Error(w, "media_file_id is required", http.StatusBadRequest)
			return
		}
		source := r.URL.Query().Get("source")
		tags, err := api.ds.MediaFileTag(r.Context()).TagsForSong(mediaFileID, source)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, tags)
	}
}

func (api *Router) allTagNames() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		source := r.URL.Query().Get("source")
		tags, err := api.ds.MediaFileTag(r.Context()).AllTagNames(source)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, tags)
	}
}

func (api *Router) tagSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p mediaFileTagPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p.MediaFileID == "" || p.TagName == "" {
			http.Error(w, "mediaFileId and tagName are required", http.StatusBadRequest)
			return
		}
		// This native REST API is the human-facing "My Tags" write path (as
		// opposed to Subsonic's setUserTag.view, which is AI Auto-Tagging's
		// own write path) - so every tag created here is source=user.
		if err := api.ds.MediaFileTag(r.Context()).TagSong(p.MediaFileID, p.TagName, model.MediaFileTagSourceUser); err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, p)
	}
}

// tagCounts returns every distinct tag name (optionally scoped to a
// source=ai|user) paired with how many songs carry it - the chip-index data
// for the AI Tags / My Tags dashboards.
func (api *Router) tagCounts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		source := r.URL.Query().Get("source")
		counts, err := api.ds.MediaFileTag(r.Context()).TagCounts(source)
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, counts)
	}
}

// tagSourceFilter is the dashboard equivalent of persistence's unexported
// mediaFileUserTagFilter, additionally scoped to one tag source (ai/user) -
// the dashboards always know which source they're browsing, unlike the
// generic user_tag song-list filter which intentionally matches either.
func tagSourceFilter(ctx context.Context, tagName, source string) Sqlizer {
	user, _ := request.UserFrom(ctx)
	cond := And{
		Expr("mft.media_file_id = media_file.id"),
		Eq{"mft.user_id": user.ID},
		Eq{"mft.tag_name": tagName},
	}
	if source != "" {
		cond = append(cond, Eq{"mft.source": source})
	}
	return Exists("media_file_tag mft", cond)
}

// tagRandomSongs mirrors genreRandomSongs (see genre.go) for the "Create
// Playlist" action on the AI Tags / My Tags dashboards - same overfetch +
// dedup + optional skip-exclusion approach, filtered by tag name/source
// instead of genre ID.
func (api *Router) tagRandomSongs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagName := chi.URLParam(r, "tag")
		if tagName == "" {
			http.Error(w, "tag is required", http.StatusBadRequest)
			return
		}
		source := r.URL.Query().Get("source")

		count := defaultGenrePlaylistTrackCount
		if c, err := strconv.Atoi(r.URL.Query().Get("count")); err == nil && c > 0 {
			count = c
		}
		if count > maxGenrePlaylistTrackCount {
			count = maxGenrePlaylistTrackCount
		}
		excludeSkipped := r.URL.Query().Get("excludeSkipped") == "true"

		ctx := r.Context()
		candidates, err := api.ds.MediaFile(ctx).GetRandom(model.QueryOptions{
			Filters: And{tagSourceFilter(ctx, tagName, source), Eq{"missing": false}},
			Max:     count * genrePlaylistOverfetchFactor,
		})
		if err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if excludeSkipped {
			filtered := candidates[:0]
			for _, mf := range candidates {
				if !mf.Skipped {
					filtered = append(filtered, mf)
				}
			}
			candidates = filtered
		}

		deduped := matcher.DeduplicateMediaFiles(candidates)
		if len(deduped) > count {
			deduped = deduped[:count]
		}

		ids := make([]string, len(deduped))
		for i, mf := range deduped {
			ids[i] = mf.ID
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, ids)
	}
}

func (api *Router) untagSong() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p mediaFileTagPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p.MediaFileID == "" || p.TagName == "" {
			http.Error(w, "mediaFileId and tagName are required", http.StatusBadRequest)
			return
		}
		if err := api.ds.MediaFileTag(r.Context()).UntagSong(p.MediaFileID, p.TagName); err != nil {
			_ = rest.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rest.RespondWithJSON(w, http.StatusOK, p)
	}
}
