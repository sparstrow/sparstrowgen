-- name: ListFavouriteModels :many
-- Oldest first: a newly starred model goes to the end, where it was put.
SELECT provider, model FROM favourite_models
WHERE user_id = $1
ORDER BY created_at, provider, model;

-- name: CountFavouriteModels :one
SELECT count(*) FROM favourite_models WHERE user_id = $1;

-- name: AddFavouriteModel :exec
-- Starring one already starred keeps its place.
INSERT INTO favourite_models (user_id, provider, model)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, provider, model) DO NOTHING;

-- name: RemoveFavouriteModel :exec
DELETE FROM favourite_models WHERE user_id = $1 AND provider = $2 AND model = $3;
