-- name: CreateFeed :one
INSERT INTO feed (id, created_at, updated_at, name, url, user_id)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2, $3)
RETURNING *;

-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT
    inserted_feed_follow.*,
    feed.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN feed ON inserted_feed_follow.feed_id = feed.id
INNER JOIN users ON inserted_feed_follow.user_id = users.id;

-- name: GetFeedByURL :one
SELECT * FROM feed WHERE url = $1;

-- name: GetFeedFollowsForUser :many
SELECT feed_follows.*, feed.name AS feed_name
FROM feed_follows
INNER JOIN feed ON feed_follows.feed_id = feed.id
WHERE feed_follows.user_id = $1;