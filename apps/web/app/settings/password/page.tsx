"use client";

import { SettingsSurface } from "@/components/settings/settings-surface";

/* The Password page's own address. /settings shows it directly on a desktop;
 * on a phone /settings is the list and this is where its first row goes
 * (docs/Decisions.md D-044). */
export default function PasswordPage() { return <SettingsSurface />; }
