package model

import "time"

type Bookmarkable struct {
	BookmarkPosition int64 `structs:"-" json:"bookmarkPosition"`
}

type BookmarkableRepository interface {
	AddBookmark(id, comment string, position int64) error
	DeleteBookmark(id string) error
	GetBookmarks() (Bookmarks, error)
	// CleanBookmarks removes any bookmark rows for this repository's item type whose referenced
	// item no longer exists - e.g. after a podcast channel (and its episodes) is torn down.
	CleanBookmarks() error
}

// Bookmark.Item holds whatever concrete type owns the bookmarked id - a MediaFile for a song, a
// PodcastEpisode for a podcast episode. It's `any` rather than a fixed type because the bookmark
// table itself is generic (item_id/item_type, not tied to one entity), and callers building an
// API response type-switch on it (see server/subsonic/bookmarks.go).
type Bookmark struct {
	Item      any       `structs:"item" json:"item"`
	Comment   string    `structs:"comment" json:"comment"`
	Position  int64     `structs:"position" json:"position"`
	ChangedBy string    `structs:"changed_by" json:"changed_by"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt"`
}

type Bookmarks []Bookmark
