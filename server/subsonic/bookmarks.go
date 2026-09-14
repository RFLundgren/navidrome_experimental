package subsonic

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

func (api *Router) GetBookmarks(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)

	songBookmarks, err := api.ds.MediaFile(ctx).GetBookmarks()
	if err != nil {
		return nil, err
	}
	episodeBookmarks, err := api.ds.PodcastEpisode(ctx).GetBookmarks()
	if err != nil {
		return nil, err
	}

	response := newResponse()
	response.Bookmarks = &responses.Bookmarks{}
	response.Bookmarks.Bookmark = make([]responses.Bookmark, 0, len(songBookmarks)+len(episodeBookmarks))
	for _, bmk := range append(songBookmarks, episodeBookmarks...) {
		var entry responses.Child
		switch item := bmk.Item.(type) {
		case model.MediaFile:
			entry = childFromMediaFile(ctx, item)
		case model.PodcastEpisode:
			entry, err = childFromPodcastEpisode(ctx, api.ds, item)
			if err != nil {
				log.Warn(ctx, "Error building bookmark entry for podcast episode", "id", item.ID, err)
				continue
			}
		default:
			continue
		}
		response.Bookmarks.Bookmark = append(response.Bookmarks.Bookmark, responses.Bookmark{
			Entry:    entry,
			Position: bmk.Position,
			Username: user.UserName,
			Comment:  bmk.Comment,
			Created:  bmk.CreatedAt,
			Changed:  bmk.UpdatedAt,
		})
	}
	return response, nil
}

func (api *Router) CreateBookmark(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	comment, _ := p.String("comment")
	position := p.Int64Or("position", 0)

	ctx := r.Context()
	entity, repo, err := api.resolveBookmarkable(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := repo.AddBookmark(id, comment, position); err != nil {
		return nil, err
	}

	if episode, ok := entity.(model.PodcastEpisode); ok {
		if err := api.recordEpisodePosition(r, episode, position); err != nil {
			log.Warn(ctx, "Error recording podcast episode position", "id", id, err)
		}
	}
	return newResponse(), nil
}

func (api *Router) DeleteBookmark(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	ctx := r.Context()
	_, repo, err := api.resolveBookmarkable(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := repo.DeleteBookmark(id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

// resolveBookmarkable finds which bookmarkable repository owns id - a song or a podcast episode
// today - and returns both the loaded entity (so a caller with type-specific follow-up work, like
// podcast completion detection, doesn't need a second lookup) and the repository to bookmark it
// through. IDs carry no embedded type marker (see model/get_entity.go), so this tries each
// bookmarkable type's Get in turn.
func (api *Router) resolveBookmarkable(ctx context.Context, id string) (any, model.BookmarkableRepository, error) {
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		return *mf, api.ds.MediaFile(ctx), nil
	}
	if ep, err := api.ds.PodcastEpisode(ctx).Get(id); err == nil {
		return *ep, api.ds.PodcastEpisode(ctx), nil
	}
	return nil, nil, model.ErrNotFound
}

func (api *Router) GetPlayQueue(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.PlayQueue(r.Context())
	pq, err := repo.RetrieveWithMediaFiles(user.ID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	if pq == nil || len(pq.Items) == 0 {
		response := newResponse()
		response.PlayQueue = &responses.PlayQueue{
			Username: user.UserName,
		}
		return response, nil
	}

	response := newResponse()
	var currentID string
	if pq.Current >= 0 && pq.Current < len(pq.Items) {
		currentID = pq.Items[pq.Current].ID
	}
	response.PlayQueue = &responses.PlayQueue{
		Entry:     slice.MapWithArg(pq.Items, r.Context(), childFromMediaFile),
		Current:   currentID,
		Position:  pq.Position,
		Username:  user.UserName,
		Changed:   pq.UpdatedAt,
		ChangedBy: pq.ChangedBy,
	}
	return response, nil
}

func (api *Router) SavePlayQueue(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids := p.Strings("id")
	currentID, _ := p.String("current")
	position := p.Int64Or("position", 0)

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	items := slice.Map(ids, func(id string) model.MediaFile {
		return model.MediaFile{ID: id}
	})

	currentIndex := 0
	for i, id := range ids {
		if id == currentID {
			currentIndex = i
			break
		}
	}

	pq := &model.PlayQueue{
		UserID:    user.ID,
		Current:   currentIndex,
		Position:  position,
		ChangedBy: client,
		Items:     items,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	repo := api.ds.PlayQueue(r.Context())
	err := repo.Store(pq)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func (api *Router) GetPlayQueueByIndex(r *http.Request) (*responses.Subsonic, error) {
	user, _ := request.UserFrom(r.Context())

	repo := api.ds.PlayQueue(r.Context())
	pq, err := repo.RetrieveWithMediaFiles(user.ID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}
	if pq == nil || len(pq.Items) == 0 {
		response := newResponse()
		response.PlayQueueByIndex = &responses.PlayQueueByIndex{
			Username: user.UserName,
		}
		return response, nil
	}

	response := newResponse()

	var index *int
	if len(pq.Items) > 0 {
		index = &pq.Current
	}

	response.PlayQueueByIndex = &responses.PlayQueueByIndex{
		Entry:        slice.MapWithArg(pq.Items, r.Context(), childFromMediaFile),
		CurrentIndex: index,
		Position:     pq.Position,
		Username:     user.UserName,
		Changed:      pq.UpdatedAt,
		ChangedBy:    pq.ChangedBy,
	}
	return response, nil
}

func (api *Router) SavePlayQueueByIndex(r *http.Request) (*responses.Subsonic, error) {
	p := req.Params(r)
	ids := p.Strings("id")

	position := p.Int64Or("position", 0)

	var err error
	var currentIndex int

	if len(ids) > 0 {
		currentIndex, err = p.Int("currentIndex")
		if err != nil || currentIndex < 0 || currentIndex >= len(ids) {
			return nil, newError(responses.ErrorMissingParameter, "missing parameter index, err: %s", err)
		}
	}

	items := slice.Map(ids, func(id string) model.MediaFile {
		return model.MediaFile{ID: id}
	})

	user, _ := request.UserFrom(r.Context())
	client, _ := request.ClientFrom(r.Context())

	pq := &model.PlayQueue{
		UserID:    user.ID,
		Current:   currentIndex,
		Position:  position,
		ChangedBy: client,
		Items:     items,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	repo := api.ds.PlayQueue(r.Context())
	err = repo.Store(pq)
	if err != nil {
		return nil, err
	}
	return newResponse(), nil
}
