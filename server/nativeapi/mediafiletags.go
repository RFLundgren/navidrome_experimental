package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) addMediaFileTagRoutes(r chi.Router) {
	r.Route("/mediaFileTag", func(r chi.Router) {
		r.Get("/", api.tagsForSong())
		r.Get("/names", api.allTagNames())
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
