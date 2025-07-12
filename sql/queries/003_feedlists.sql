-- name: ListFeedsWithCreators :many
SELECT feed.name AS feed_name, feed.url, users.name AS user_name
FROM feed
LEFT JOIN users ON feed.user_id = users.id;