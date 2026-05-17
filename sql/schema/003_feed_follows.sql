-- +goose Up
CREATE TABLE feeds_follows(
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    feed_id UUID NOT NULL,
    UNIQUE(user_id, feed_id),
    CONSTRAINT fk_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_feed
        FOREIGN KEY(feed_id)
        REFERENCES feeds(id)
        ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds_follows;