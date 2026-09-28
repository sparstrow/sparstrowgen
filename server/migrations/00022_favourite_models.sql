-- +goose Up
-- The models a person has starred in the agent and model picker, from any agent
-- (docs/design/prototypes/Chat/model-picker.handoff.md). Per account, like
-- appearance and the profile, so the list follows the person to every browser.
--
-- A row per starred model rather than an array on users: `users` is read on
-- every request to resolve the session (D-048), and a row carries its own time,
-- which is the order the list is shown in.
--
-- The provider and model are the ids the daemon reports, kept as text and never
-- checked against a list: a newer daemon may offer a model this server has
-- never heard of, and a model the computer stops offering is hidden by the
-- page, not deleted here, so it comes back if the model does.
CREATE TABLE favourite_models (
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider   text        NOT NULL,
    model      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, provider, model)
);

-- +goose Down
DROP TABLE favourite_models;
