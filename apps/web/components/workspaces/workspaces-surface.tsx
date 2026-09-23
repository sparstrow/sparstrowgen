"use client";

import { useWorkspace, useWorkspaces } from "@/lib/queries";
import { SettingsShell } from "@/components/settings/settings-shell";
import {
  SettingsCard,
  SettingsPage,
  SettingsSection,
} from "@/components/settings/settings-rows";
import { WorkspaceFields } from "@/components/workspaces/workspace-fields";
import { WorkspaceMachines } from "@/components/workspaces/workspace-machines";

/* Settings → Workspaces. The same editor first-run setup's second step shows,
   so there is one place workspaces are made and named rather than two that
   drift (docs/Decisions.md D-050).

   There is no delete. A workspace holds conversations, and removing one would
   be the most destructive thing in this app — it needs its own design, with
   what it is about to take said out loud beforehand. Until then the honest
   answer is that it is not offered, which is what the note below says. */

export function WorkspacesSurface() {
  const workspaces = useWorkspaces();
  const count = workspaces.data?.length ?? 0;
  // The workspace this browser is in. Switching with the "Open" buttons above
  // changes it, so the computers below are always the ones for the workspace
  // whose name is on the section heading.
  const open = useWorkspace();

  return (
    <SettingsShell current="workspaces">
      <SettingsPage
        title="Workspaces"
        description="Separate areas of work inside your account. Conversations, searches and folders stay inside the workspace they belong to."
      >
        {/* No `aside` here: the pane leaves this column narrow, and a sentence
            beside the heading collided with it. It belongs with the fields it
            describes anyway. */}
        <SettingsSection title={count === 1 ? "Your workspace" : "Your workspaces"}>
          <SettingsCard className="p-5">
            <div className="space-y-4">
              <p className="text-xs text-muted-foreground">
                Click a name to change it. Opening one switches this browser; another tab or device
                stays where it was.
              </p>
              <WorkspaceFields />
            </div>
          </SettingsCard>
        </SettingsSection>

        {/* Which computers the workspace you are IN may use. One workspace at a
            time, the one on screen, rather than a grid of every workspace
            against every computer — the question people arrive with is "can
            this one use my work laptop", and a matrix answers a question
            nobody asked. The other workspaces are one switch away. */}
        <SettingsSection title={open ? `Computers in ${open.name}` : "Computers"}>
          <SettingsCard>
            {open ? (
              <WorkspaceMachines workspaceId={open.id} />
            ) : (
              <p className="px-4 py-3 text-sm text-muted-foreground">
                Open a workspace above to choose its computers.
              </p>
            )}
          </SettingsCard>
        </SettingsSection>

        <SettingsSection title="What is shared, and what is not">
          <SettingsCard className="p-5">
            {/* The card's own content is a divided list of rows; this is
                prose, so it carries its own spacing. */}
            <div className="space-y-3 text-sm text-muted-foreground">
            <p>
              A computer belongs to your account, not to one workspace: you pair it once and it is
              offered everywhere. Above is where you take it out of the workspaces that do not need
              it — and a workspace runs its work only on the computers it still has.
            </p>
            <p>
              Deleting a workspace is not possible yet. It would take every conversation in it with
              it, so it needs to say exactly what it is about to remove before it is offered.
            </p>
            </div>
          </SettingsCard>
        </SettingsSection>
      </SettingsPage>
    </SettingsShell>
  );
}
