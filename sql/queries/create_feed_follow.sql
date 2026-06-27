-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (
        id,
        created_at,
        updated_at,
        user_id,
        feed_id
    )
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)
SELECT
    iff.*,
    users.name AS user_name,
    feeds.name AS feed_name
FROM inserted_feed_follow iff
JOIN users ON iff.user_id = users.id
JOIN feeds ON iff.feed_id = feeds.id;

-- name: GetFeedByURL :one
SELECT *
FROM feeds
WHERE url = $1;

-- name: GetFeedFollowsForUser :many
SELECT
    ff.*,
    users.name AS user_name,
    feeds.name AS feed_name
FROM feed_follows ff
JOIN users ON ff.user_id = users.id
JOIN feeds ON ff.feed_id = feeds.id
WHERE ff.user_id = $1;