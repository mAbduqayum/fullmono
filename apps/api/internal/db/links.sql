-- name: CreateLink :one
INSERT INTO links (code, url) VALUES ($1, $2)
RETURNING *;

-- name: ListLinks :many
SELECT * FROM links
ORDER BY created_at DESC
LIMIT $1;

-- name: VisitLink :one
UPDATE links SET clicks = clicks + 1 WHERE code = $1
RETURNING url;

-- name: CountMatchingLinks :one
SELECT count(*) FROM links
WHERE code ILIKE @pattern OR url ILIKE @pattern;

-- name: SearchLinks :many
-- id breaks ties between links made in the same instant, so pages never overlap.
SELECT * FROM links
WHERE code ILIKE @pattern OR url ILIKE @pattern
ORDER BY created_at DESC, id DESC
LIMIT @page_size OFFSET @skip;

-- name: DeleteLink :execrows
DELETE FROM links WHERE code = $1;

-- name: CountLinksAndClicks :one
-- sum() of bigints is a numeric, and NULL over no rows at all.
SELECT count(*) AS links, coalesce(sum(clicks), 0)::bigint AS clicks FROM links;

-- name: TopLinks :many
SELECT * FROM links WHERE clicks > 0
ORDER BY clicks DESC, created_at DESC
LIMIT $1;
