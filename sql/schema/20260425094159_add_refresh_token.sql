-- +goose Up
CREATE TABLE refresh_tokens(
    token TEXT PRIMARY KEY,
    created_at Timestamp,
    updated_at Timestamp,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
    expires_at Timestamp not null,
    revoked_at Timestamp
);

-- +goose Down
DELETE FROM refresh_tokens;
