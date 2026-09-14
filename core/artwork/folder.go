package artwork

import (
	"context"
	"fmt"
	"image"
	"path"
	"slices"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
)

// FolderGridSamples is how many albums resolveFolder samples to build the generated grid,
// mirroring PlaylistGridSamples.
const FolderGridSamples = 4

// resolveFolder walks conf.Server.CoverArtPriority the same way a library folder's cover art
// always has, falling back to a generated 2x2 grid sampled from the folder's own albums (by track
// order, not randomly - a folder's contents are usually already meaningfully ordered).
func (r *resolver) resolveFolder(ctx context.Context, folderID string) (resolution, error) {
	f, err := r.ds.Folder(ctx).Get(folderID)
	if err != nil {
		return resolution{}, err
	}
	lib, err := loadLibraryView(ctx, r.ds, f.LibraryID)
	if err != nil {
		return resolution{}, err
	}

	var imgFiles []string
	rel := strings.TrimPrefix(path.Join(f.Path, f.Name), "/")
	for _, img := range f.ImageFiles {
		imgFiles = append(imgFiles, path.Join(rel, img))
	}
	slices.SortFunc(imgFiles, compareImageFiles)

	chain := chainState{trace: traceFrom(ctx)}
	priority := strings.ToLower(conf.Server.CoverArtPriority)
	for pattern := range strings.SplitSeq(priority, ",") {
		pattern = strings.TrimSpace(pattern)
		if err := ctx.Err(); err != nil {
			return resolution{}, err
		}
		var res resolution
		var ok bool
		switch {
		case pattern == "embedded":
			res, ok = r.resolveFolderEmbedded(ctx, lib, *f)
		case pattern == externalCandidate:
			chain.record(pattern, OutcomeSkipped, "no external provider for folder artwork")
			continue
		case len(imgFiles) == 0:
			chain.record(pattern, OutcomeSkipped, "no images in folder")
			continue
		default:
			res, ok = resolveFolderSource(lib, fromExternalFile(ctx, lib.FS, imgFiles, pattern))
		}
		if res, ok = chain.try(pattern, res, ok); ok {
			return res, nil
		}
	}

	res, err := r.resolveFolderGrid(ctx, *f)
	if err != nil {
		return resolution{}, err
	}
	if res.reader != nil {
		return res, nil
	}
	return chain.exhausted(), nil
}

// resolveFolderEmbedded tries the first non-missing track's own embedded art as the folder's art.
func (r *resolver) resolveFolderEmbedded(ctx context.Context, lib libraryView, f model.Folder) (resolution, bool) {
	tracks, err := r.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: Eq{"folder_id": f.ID, "media_file.missing": false},
		Max:     1,
	})
	if err != nil || len(tracks) == 0 {
		return resolution{}, false
	}
	return resolveEmbedded(ctx, lib, r.ffmpeg, tracks[0].Path)
}

// resolveFolderGrid samples up to FolderGridSamples distinct albums (by track order within the
// folder hierarchy) and tiles their own covers into a 2x2 grid, the same fallback a playlist with
// no cover of its own gets.
func (r *resolver) resolveFolderGrid(ctx context.Context, f model.Folder) (resolution, error) {
	tracks, err := r.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: Eq{"folder_id_recursive": f.ID, "media_file.missing": false},
		Max:     100, // enough tracks to find diverse albums
	})
	if err != nil {
		return resolution{}, err
	}

	seen := make(map[string]bool)
	var albumIDs []string
	for _, t := range tracks {
		if t.AlbumID == "" || seen[t.AlbumID] {
			continue
		}
		seen[t.AlbumID] = true
		albumIDs = append(albumIDs, t.AlbumID)
		if len(albumIDs) == FolderGridSamples {
			break
		}
	}
	if len(albumIDs) == 0 {
		return resolution{}, nil
	}

	var tileErr error
	tiles := make([]image.Image, 0, FolderGridSamples)
	for _, albumID := range albumIDs {
		res, err := r.resolveAlbum(ctx, albumID)
		if err != nil {
			if tileErr == nil {
				tileErr = err
			}
			continue
		}
		if res.reader == nil {
			continue
		}
		tile, decErr := decodeTile(res.reader)
		res.reader.Close()
		if decErr == nil {
			tiles = append(tiles, tile)
		}
	}
	if len(tiles) == 0 {
		if tileErr != nil {
			return resolution{}, fmt.Errorf("resolveFolderGrid: sampled album art failed: %w", tileErr)
		}
		return resolution{}, nil
	}
	switch len(tiles) {
	case 2:
		tiles = append(tiles, tiles[1], tiles[0])
	case 3:
		tiles = append(tiles, tiles[0])
	}
	grid, err := assembleTiles(tiles)
	if err != nil {
		return resolution{}, nil //nolint:nilerr // encode failure is a soft "no image", not a resolution error
	}
	return resolution{reader: grid, source: "generated"}, nil
}
