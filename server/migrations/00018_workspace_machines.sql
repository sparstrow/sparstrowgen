-- +goose Up
-- Which of an account's computers a workspace may use.
--
-- The owner, the day after workspaces landed: "I need a way and a settings to
-- be able to add the machine to the workspace. If I have multiple machine in my
-- account. I need to choose which machine needs added to that workspace or vice
-- versa whick workspace needs to added to the machines."
--
-- So it is many-to-many and editable from either end — one machine in several
-- workspaces, one workspace with several machines — which is why this is its
-- own table rather than a column on either side.
--
-- A computer still BELONGS to the account (docs/Decisions.md D-031, D-050).
-- This says where it is offered, not who owns it. That distinction is what
-- keeps pairing a once-per-computer job rather than a once-per-workspace one.
CREATE TABLE workspace_machines (
    workspace_id uuid        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    machine_id   uuid        NOT NULL REFERENCES machines (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, machine_id)
);

-- The question asked on every turn: which computers may this workspace use.
CREATE INDEX workspace_machines_workspace_idx ON workspace_machines (workspace_id);
-- And its mirror, for the machine's own page: which workspaces offer it.
CREATE INDEX workspace_machines_machine_idx ON workspace_machines (machine_id);

-- ON DELETE CASCADE on both, unlike everything else in this schema, and that is
-- deliberate: a row here is an ASSIGNMENT, not work. Losing it when either side
-- goes is correct, and the rule it would otherwise break (D-009, deleting must
-- not silently take work with it) is about transcripts.

-- Everything that already exists is assigned everywhere it could be.
--
-- The alternative — start empty and make people assign — would have taken every
-- working account's computer away at deploy time and left Chat unable to send
-- until somebody found a settings page they had never needed before. The
-- default is that a computer is available in every workspace you work in, which
-- is what was true a minute before this migration ran; narrowing it is the new
-- deliberate act.
INSERT INTO workspace_machines (workspace_id, machine_id)
SELECT w.id, m.id
FROM workspaces w
JOIN workspace_members wm ON wm.workspace_id = w.id
JOIN machines m ON m.user_id = wm.user_id
WHERE m.approved_at IS NOT NULL AND m.revoked_at IS NULL
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE workspace_machines;
