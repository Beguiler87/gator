-- name: ListFeedsWithCreators :many
SELECT feed.name AS feed_name, feed.url, users.name AS user_name
FROM feed
LEFT JOIN users ON feed.user_id = users.id;

-- name: UnfollowFeed :exec
DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;

-- name: GetFeed :one
SELECT id FROM feed
WHERE url = $1;

-- name: MarkFeedFetched :exec
UPDATE feed
SET last_fetched_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feed
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;