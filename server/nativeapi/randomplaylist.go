package nativeapi

import (
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/matcher"
	"github.com/navidrome/navidrome/model"
)

const (
	defaultGenrePlaylistTrackCount = 50
	maxGenrePlaylistTrackCount     = 500
	// genrePlaylistOverfetchFactor over-fetches candidates so there's still enough left after
	// skip-exclusion, dedup and the per-artist cap to reach the requested count. Every step below
	// is best-effort against this fixed pool - there's no retry-until-full loop, so a narrow
	// enough combination of options (e.g. a low maxPerArtist on a genre with few artists) can
	// return fewer than the requested count.
	genrePlaylistOverfetchFactor = 3
)

// randomPlaylistOptions bundles the "Create Playlist" dialog's optional narrowing knobs, shared
// by genreRandomSongs and tagRandomSongs.
type randomPlaylistOptions struct {
	count             int
	excludeSkipped    bool
	excludeDuplicates bool
	maxPerArtist      int // 0 = unlimited
}

// parseRandomPlaylistOptions reads the dialog's query params, shared by genreRandomSongs and
// tagRandomSongs. excludeDuplicates defaults to true (the previous, unconditional behavior) so
// existing callers that omit it see no change; maxPerArtist defaults to 0 (unlimited).
func parseRandomPlaylistOptions(r *http.Request) randomPlaylistOptions {
	count := defaultGenrePlaylistTrackCount
	if c, err := strconv.Atoi(r.URL.Query().Get("count")); err == nil && c > 0 {
		count = c
	}
	if count > maxGenrePlaylistTrackCount {
		count = maxGenrePlaylistTrackCount
	}

	maxPerArtist := 0
	if m, err := strconv.Atoi(r.URL.Query().Get("maxPerArtist")); err == nil && m > 0 {
		maxPerArtist = m
	}

	return randomPlaylistOptions{
		count:             count,
		excludeSkipped:    r.URL.Query().Get("excludeSkipped") == "true",
		excludeDuplicates: r.URL.Query().Get("excludeDuplicates") != "false",
		maxPerArtist:      maxPerArtist,
	}
}

// buildRandomPlaylist applies excludeSkipped, dedup and the per-artist cap (in that order) to an
// already-overfetched candidate pool, then truncates to opts.count and returns the surviving IDs.
func buildRandomPlaylist(candidates model.MediaFiles, opts randomPlaylistOptions) []string {
	if opts.excludeSkipped {
		filtered := candidates[:0]
		for _, mf := range candidates {
			if !mf.Skipped {
				filtered = append(filtered, mf)
			}
		}
		candidates = filtered
	}

	if opts.excludeDuplicates {
		candidates = matcher.DeduplicateMediaFiles(candidates)
	}

	if opts.maxPerArtist > 0 {
		perArtist := make(map[string]int, len(candidates))
		filtered := candidates[:0]
		for _, mf := range candidates {
			if perArtist[mf.ArtistID] >= opts.maxPerArtist {
				continue
			}
			perArtist[mf.ArtistID]++
			filtered = append(filtered, mf)
		}
		candidates = filtered
	}

	if len(candidates) > opts.count {
		candidates = candidates[:opts.count]
	}

	ids := make([]string, len(candidates))
	for i, mf := range candidates {
		ids[i] = mf.ID
	}
	return ids
}
