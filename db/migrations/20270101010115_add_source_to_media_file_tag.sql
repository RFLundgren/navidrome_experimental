-- +goose Up
-- +goose StatementBegin
ALTER TABLE media_file_tag ADD COLUMN source varchar(20) not null default 'user';
-- +goose StatementEnd
-- +goose StatementBegin
-- Backfill existing rows by tag_name shape: only AI Auto-Tagging ever writes
-- the genre:/mood:/language: prefixed convention, so any row matching it
-- predates this column and was AI-written; everything else (including tags
-- added by hand via the existing "Edit Tags" dialog before this migration)
-- correctly stays 'user', the column's default.
UPDATE media_file_tag SET source = 'ai'
WHERE tag_name LIKE 'genre:%' OR tag_name LIKE 'mood:%' OR tag_name LIKE 'language:%';
-- +goose StatementEnd

-- +goose Down
