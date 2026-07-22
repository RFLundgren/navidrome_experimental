package model

import "time"

// Tag source values distinguish tags written by the AI Auto-Tagging plugin
// (via the Subsonic setUserTag.view API) from tags a human added themselves
// (via the native REST /mediaFileTag API) - see MediaFileTagRepository.
const (
	MediaFileTagSourceAI   = "ai"
	MediaFileTagSourceUser = "user"
)

type MediaFileTag struct {
	UserID      string    `structs:"user_id"       json:"userId"`
	MediaFileID string    `structs:"media_file_id" json:"mediaFileId"`
	TagName     string    `structs:"tag_name"      json:"tagName"`
	Source      string    `structs:"source"        json:"source"`
	CreatedAt   time.Time `structs:"created_at"    json:"createdAt"`
}

// MediaFileTagRepository stores per-user tags on media files. Every tag has
// a source (MediaFileTagSourceAI or MediaFileTagSourceUser) recording who/
// what wrote it. TagsForSong, AllTagNames, and SongIDsForTag take a source
// filter; pass "" to match tags of any source.
type MediaFileTagRepository interface {
	TagSong(mediaFileID, tagName, source string) error
	UntagSong(mediaFileID, tagName string) error
	TagsForSong(mediaFileID, source string) ([]string, error)
	AllTagNames(source string) ([]string, error)
	SongIDsForTag(tagName, source string) ([]string, error)
}
