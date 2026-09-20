"use client";

import { SettingsSurface } from "@/components/settings/settings-surface";
import { SettingsShell } from "@/components/settings/settings-shell";
import { useIsMobile } from "@/hooks/use-mobile";

/* The section's front door. On a desktop the pane and a page are shown side by
 * side, so landing here opens the first page rather than half an empty screen.
 * On a phone the pane IS the screen (docs/Decisions.md D-044), so this is the
 * list and the page it used to be lives at /settings/password. */
export default function SettingsPage() {
  const isMobile = useIsMobile();
  // SettingsSurface brings its own shell, so this only chooses which of the two
  // screens /settings is.
  if (isMobile) return <SettingsShell current={null}>{null}</SettingsShell>;
  return <SettingsSurface />;
}
