"use client";

import { useProfile, useSession } from "@/lib/queries";
import { Skeleton } from "@/components/ui/skeleton";
import { Status } from "@/components/ui/status";
import { Button } from "@/components/ui/button";
import { SettingsShell } from "@/components/settings/settings-shell";
import { SettingsCard, SettingsPage, SettingsRow, SettingsSection } from "@/components/settings/settings-rows";
import { ProfileFields } from "@/components/settings/profile-fields";

/* Settings → Account. The same fields first-run setup shows, so there is one
   profile editor rather than two that drift.

   The email address sits here too, as a row that cannot be edited. Changing the
   address is a different job — it needs the new one proved, like registration —
   and showing it greyed out with no explanation would read as broken. */

export function AccountSurface() {
  const session = useSession();
  const profile = useProfile();

  return (
    <SettingsShell current="account">
      <SettingsPage
        title="Account"
        description="Who you are in sparstrowgen. This follows your account, so it is the same in every browser you sign in to."
      >
        <SettingsSection title="Profile">
          <SettingsCard className="p-5">
            {profile.isPending ? (
              <div className="space-y-5" aria-busy="true">
                <div className="flex items-center gap-4">
                  <Skeleton className="size-16 rounded-full" />
                  <Skeleton className="h-8 w-40" />
                </div>
                <Skeleton className="h-9 w-full" />
                <Skeleton className="h-20 w-full" />
              </div>
            ) : profile.isError ? (
              <div className="flex flex-col items-start gap-3">
                <Status tone="danger" quiet>
                  Your profile could not be loaded
                </Status>
                <p className="text-sm text-muted-foreground">
                  Nothing has changed. {(profile.error as Error).message}
                </p>
                <Button variant="outline" size="sm" onClick={() => void profile.refetch()}>
                  Try again
                </Button>
              </div>
            ) : (
              <ProfileFields />
            )}
          </SettingsCard>
        </SettingsSection>

        <SettingsSection title="Sign in">
          <SettingsCard>
            <SettingsRow
              label="Email address"
              description="How you sign in, and where account emails are sent. Changing it needs the new address proved, so it is not editable here yet."
            >
              <span className="truncate text-sm text-muted-foreground" title={session.data?.email}>
                {session.data?.email ?? "—"}
              </span>
            </SettingsRow>
            <SettingsRow label="Password" description="Change it from the Password page.">
              <Button variant="outline" size="sm" render={<a href="/settings/password" />} nativeButton={false}>
                Change password
              </Button>
            </SettingsRow>
          </SettingsCard>
        </SettingsSection>
      </SettingsPage>
    </SettingsShell>
  );
}
