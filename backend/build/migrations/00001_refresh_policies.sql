-- +goose Up
CREATE TABLE IF NOT EXISTS materialized_view_refresh_policies (
    policy_key TEXT PRIMARY KEY,
    policy TEXT NOT NULL
);

-- +goose Down
DROP TABLE materialized_view_refresh_policies;
