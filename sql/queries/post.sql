-- name: CreatePost :one
INSERT INTO posts(
    id, created_at, updated_at, title, url, description, published_at, feed_id
) VALUES(
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetPostForUser :many
SELECT posts.*
FROM posts
INNER JOIN feeds_follows 
    ON posts.feed_id = feeds_follows.feed_id
WHERE feeds_follows.user_id = $1
ORDER BY published_at DESC
LIMIT $2;