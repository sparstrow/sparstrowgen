"use client";

import { useState } from "react";
import { KeyRound, LogOut, MonitorSmartphone, UserRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSession, useSignOut } from "@/lib/queries";
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
  const signOut = useSignOut();

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
          <UserRound className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          {/* Not a menu item — there is nothing to click. It is a label saying
              whose account this is. */}
          <div className="flex items-center gap-2 px-2 py-1.5">
            <UserRound className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            <span className="truncate text-sm" title={session.data?.email}>
              {session.data?.email ?? "Signed in"}
            </span>
          </div>
          <DropdownMenuSeparator />

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
