"use client";

import { LogOut, MonitorSmartphone } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSignOut } from "@/lib/queries";

/* Signing out.
 *
 * Two entries rather than one, because they answer different worries. "Sign
 * out" is leaving this browser; "everywhere" is for the worry that something
 * else is still signed in — a phone left somewhere, a machine sold. The second
 * is the one that is useless if you have to work out how to do it under
 * pressure, which is why it is here rather than in a settings page nobody has
 * opened.
 */
export function SignOutMenu() {
  const signOut = useSignOut();

  return (
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
        <LogOut className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuItem onClick={() => signOut.mutate(false)}>
          <LogOut className="size-4" />
          Sign out
        </DropdownMenuItem>
        <DropdownMenuSeparator />
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
  );
}
