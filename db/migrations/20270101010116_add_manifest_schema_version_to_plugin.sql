-- +goose Up
-- +goose StatementBegin
-- Existing rows default to 0, which is below any real CurrentManifestSchemaVersion
-- (see plugins.CurrentManifestSchemaVersion), so the next plugin sync re-extracts
-- every already-installed plugin's manifest at least once after this migration -
-- picking up any manifest field added since that plugin was first scanned.
ALTER TABLE plugin ADD COLUMN manifest_schema_version integer not null default 0;
-- +goose StatementEnd

-- +goose Down
