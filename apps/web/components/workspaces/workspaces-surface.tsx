"use client";

import { useWorkspaces } from "@/lib/queries";
import { SettingsShell } from "@/components/settings/settings-shell";
import {
  SettingsCard,
  SettingsPage,
  SettingsSection,
} from "@/components/settings/settings-rows";
import { WorkspaceFields } from "@/components/workspaces/workspace-fields";

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

        <SettingsSection title="What is shared, and what is not">
          <SettingsCard className="p-5">
            {/* The card's own content is a divided list of rows; this is
                prose, so it carries its own spacing. */}
            <div className="space-y-3 text-sm text-muted-foreground">
            <p>
              Your computers belong to your account rather than to one workspace, so a computer you
              have connected is available in every workspace you work in. You do not pair it again.
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
