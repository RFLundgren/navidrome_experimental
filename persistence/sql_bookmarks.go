package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

const bookmarkTable = "bookmark"

func (r sqlRepository) withBookmark(query SelectBuilder, idField string) SelectBuilder {
	userID := loggedUser(r.ctx).ID
	if userID == invalidUserId {
		return query
	}
	return query.
		LeftJoin("bookmark on (" +
			"bookmark.item_id = " + idField +
			" AND bookmark.user_id = '" + userID + "')").
		Columns("coalesce(position, 0) as bookmark_position")
}

func (r sqlRepository) bmkID(itemID ...string) And {
	return And{
		Eq{bookmarkTable + ".user_id": loggedUser(r.ctx).ID},
		Eq{bookmarkTable + ".item_type": r.tableName},
		Eq{bookmarkTable + ".item_id": itemID},
	}
}

func (r sqlRepository) bmkUpsert(itemID, comment string, position int64) error {
	client, _ := request.ClientFrom(r.ctx)
	user, _ := request.UserFrom(r.ctx)
	values := map[string]any{
		"comment":    comment,
		"position":   position,
		"updated_at": time.Now(),
		"changed_by": client,
	}

	upd := Update(bookmarkTable).Where(r.bmkID(itemID)).SetMap(values)
	c, err := r.executeSQL(upd)
	if err == nil {
		log.Debug(r.ctx, "Updated bookmark", "id", itemID, "user", user.UserName, "position", position, "comment", comment)
	}
	if c == 0 || errors.Is(err, sql.ErrNoRows) {
		values["user_id"] = user.ID
		values["item_type"] = r.tableName
		values["item_id"] = itemID
		values["created_at"] = time.Now()
		values["updated_at"] = time.Now()
		ins := Insert(bookmarkTable).SetMap(values)
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
		log.Debug(r.ctx, "Added bookmark", "id", itemID, "user", user.UserName, "position", position, "comment", comment)
	}

	return err
}

func (r sqlRepository) AddBookmark(id, comment string, position int64) error {
	user, _ := request.UserFrom(r.ctx)
	err := r.bmkUpsert(id, comment, position)
	if err != nil {
		log.Error(r.ctx, "Error adding bookmark", "id", id, "user", user.UserName, "position", position, "comment", comment)
	}
	return err
}

func (r sqlRepository) DeleteBookmark(id string) error {
	user, _ := request.UserFrom(r.ctx)
	del := Delete(bookmarkTable).Where(r.bmkID(id))
	_, err := r.executeSQL(del)
	if err != nil {
		log.Error(r.ctx, "Error removing bookmark", "id", id, "user", user.UserName)
	}
	return err
}

type bookmark struct {
	UserID    string    `json:"user_id"`
	ItemID    string    `json:"item_id"`
	ItemType  string    `json:"item_type"`
	Comment   string    `json:"comment"`
	Position  int64     `json:"position"`
	ChangedBy string    `json:"changed_by"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// bookmarkedItemIDs returns the ids of this repository's own items (media files, podcast
// episodes, ...) that the current user has bookmarked. Querying from r.tableName rather than the
// bookmark table directly is deliberate: it's what excludes a stale bookmark left behind by a
// since-deleted item (see cleanBookmarks, which sweeps those up eventually, but reads shouldn't
// have to wait for that).
func (r sqlRepository) bookmarkedItemIDs() ([]string, error) {
	idField := r.tableName + ".id"
	sq := r.newSelect().Columns(idField)
	sq = r.withBookmark(sq, idField).Where(NotEq{bookmarkTable + ".item_id": nil})
	var rows []struct{ ID string }
	if err := r.queryAll(sq, &rows); err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids, nil
}

// bookmarksByID returns this repository's own raw bookmark rows (comment, position, timestamps),
// keyed by item id. Callers combine this with their own typed GetAll (which already knows how to
// decode their table's columns) to assemble model.Bookmark values - see
// mediaFileRepository.GetBookmarks / podcastEpisodeRepository.GetBookmarks. This split exists
// because a single query decoding into one concrete type (the previous approach, hardcoded to
// dbMediaFile) can't serve every bookmarkable entity type generically.
func (r sqlRepository) bookmarksByID(itemIDs []string) (map[string]bookmark, error) {
	sq := Select("*").From(bookmarkTable).Where(r.bmkID(itemIDs...))
	var bmks []bookmark
	if err := r.queryAll(sq, &bmks); err != nil {
		return nil, err
	}
	byID := make(map[string]bookmark, len(bmks))
	for _, bmk := range bmks {
		byID[bmk.ItemID] = bmk
	}
	return byID, nil
}

func (r sqlRepository) reassignBookmark(prevID, newID string) error {
	upd := Expr("update or ignore "+bookmarkTable+" set item_id = ? where item_type = ? and item_id = ?",
		newID, r.tableName, prevID)
	_, err := r.executeSQL(upd)
	return err
}

func (r sqlRepository) cleanBookmarks() error {
	del := Delete(bookmarkTable).Where(Eq{"item_type": r.tableName}).Where("item_id not in (select id from " + r.tableName + ")")
	c, err := r.executeSQL(del)
	if err != nil {
		return fmt.Errorf("error cleaning up %s bookmarks: %w", r.tableName, err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Clean-up bookmarks", "totalDeleted", c, "itemType", r.tableName)
	}
	return nil
}

// CleanBookmarks is the exported counterpart to cleanBookmarks, for callers outside package
// persistence (e.g. core/podcasts, tearing down an orphaned podcast channel's episodes) that only
// have the model.BookmarkableRepository interface to work with, not the unexported concrete type.
func (r sqlRepository) CleanBookmarks() error {
	return r.cleanBookmarks()
}
