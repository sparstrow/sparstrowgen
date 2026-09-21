"use client";

import { useState } from "react";
import Link from "next/link";
import { KeyRound, LogOut, MonitorSmartphone, UserRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useProfile, useSession, useSignOut } from "@/lib/queries";
import { api } from "@/lib/api";
import { initials } from "@/lib/avatar";
import { Avatar } from "@/components/ui/avatar";
import { ChangePassword } from "./change-password";

/* The account, which is one account.
 *
 * The email at the top is not decoration: it is the only place the app ever
 * says WHO is signed in, and the question it answers — "am I looking at my own
 * deployment or somebody else's" — is one you only think to ask when the answer
 * matters.
 *
 * Two sign-out entries rather than one, because they answer different worries.
 * "Sign out" is leaving this browser; "everywhere" is for the worry that
 * something else is still signed in — a phone left somewhere, a machine sold.
 * The second is useless if you have to work out how to do it under pressure,
 * which is why it is here rather than in a settings page nobody has opened.
 */
export function AccountMenu() {
  const [changing, setChanging] = useState(false);
  const session = useSession();
  const profile = useProfile();
  const signOut = useSignOut();

  // The picture if there is one, otherwise initials. Never an empty circle:
  // that reads as something failing to load rather than as a choice nobody
  // made. The email is the fallback for the letters, so this says something
  // even before a name is set.
  const src = api.avatarUrl(profile.data?.avatarUpdatedAt);
  const letters = initials(profile.data?.displayName, session.data?.email);
  const name = profile.data?.displayName?.trim() || session.data?.email;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              aria-label="Account"
              disabled={signOut.isPending}
            />
          }
        >
          <Avatar src={src} initials={letters} className="size-7" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          {/* Not a menu item — there is nothing to click. It is a label saying
              whose account this is. */}
          <div className="flex items-center gap-2 px-2 py-1.5">
            <Avatar src={src} initials={letters} className="size-7" />
            <span className="flex min-w-0 flex-col">
              <span className="truncate text-sm" title={name}>
                {name ?? "Signed in"}
              </span>
              {/* Only when the name is not already the address, so the two
                  lines never say the same thing twice. */}
              {profile.data?.displayName?.trim() ? (
                <span className="truncate text-xs text-muted-foreground" title={session.data?.email}>
                  {session.data?.email}
                </span>
              ) : null}
            </span>
          </div>
          <DropdownMenuSeparator />

          <DropdownMenuItem render={<Link href="/settings/account" />}>
            <UserRound className="size-4" />
            Edit profile
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setChanging(true)}>
            <KeyRound className="size-4" />
            Change password
          </DropdownMenuItem>
          <DropdownMenuSeparator />

          <DropdownMenuItem onClick={() => signOut.mutate(false)}>
            <LogOut className="size-4" />
            Sign out
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => signOut.mutate(true)}>
            <MonitorSmartphone className="size-4" />
            <span className="flex flex-col items-start">
              Sign out everywhere
              <span className="text-xs text-muted-foreground">
                Ends every other device too
              </span>
            </span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <ChangePassword open={changing} onOpenChange={setChanging} />
    </>
  );
}
