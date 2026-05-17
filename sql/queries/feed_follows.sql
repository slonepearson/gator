-- name: AddFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feeds_follows(
        id, created_at, updated_at, user_id, feed_id
    ) VALUES (
        $1, $2, $3, $4, $5
    )
    RETURNING *
)

SELECT 
    inserted_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN users ON inserted_feed_follow.user_id = users.id
INNER JOIN feeds ON inserted_feed_follow.feed_id = feeds.id;

-- name: GetFeedFollowsForUser :many
SELECT 
    feeds_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feeds_follows
INNER JOIN users ON feeds_follows.user_id = users.id
INNER JOIN feeds ON feeds_follows.feed_id = feeds.id
WHERE feeds_follows.user_id = $1;