package model

import (
	"context"
	"errors"
)

// TODO: Should the type be encoded in the ID?
func GetEntityByID(ctx context.Context, ds DataStore, id string) (any, error) {
	entity, _, err := getEntity(ctx, ds, id)
	return entity, err
}

// GetEntityKindByID resolves a bare entity id to its artwork Kind, searching the same tables as
// GetEntityByID. It reports ErrNotFound when no entity owns the id. Podcast episodes have no
// artwork Kind of their own (they inherit the parent channel's artwork), so they resolve with a
// zero Kind{} rather than an error - the id is real, it just isn't independently art-bearing.
func GetEntityKindByID(ctx context.Context, ds DataStore, id string) (Kind, error) {
	_, kind, err := getEntity(ctx, ds, id)
	return kind, err
}

func getEntity(ctx context.Context, ds DataStore, id string) (any, Kind, error) {
	getters := []struct {
		kind Kind
		get  func() (any, error)
	}{
		{KindArtistArtwork, func() (any, error) { return ds.Artist(ctx).Get(id) }},
		{KindAlbumArtwork, func() (any, error) { return ds.Album(ctx).Get(id) }},
		{KindPlaylistArtwork, func() (any, error) { return ds.Playlist(ctx).Get(id) }},
		{KindMediaFileArtwork, func() (any, error) { return ds.MediaFile(ctx).Get(id) }},
		{KindRadioArtwork, func() (any, error) { return ds.Radio(ctx).Get(id) }},
		{KindFolderArtwork, func() (any, error) { return ds.Folder(ctx).Get(id) }},
		{KindPodcastChannelArtwork, func() (any, error) { return ds.PodcastChannel(ctx).Get(id) }},
		{Kind{}, func() (any, error) { return ds.PodcastEpisode(ctx).Get(id) }},
	}
	for _, g := range getters {
		entity, err := g.get()
		if err == nil {
			return entity, g.kind, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, Kind{}, err
		}
	}
	return nil, Kind{}, ErrNotFound
}
