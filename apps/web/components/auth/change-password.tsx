"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useChangePassword } from "@/lib/queries";
import { Field, FormError } from "./shell";

/* Changing the password, from inside the app.
 *
 * A dialog rather than a settings page, and rather than something inline: this
 * is the whole of the account surface, a page for one form would be a page the
 * owner visits once, and the task genuinely wants protected focus — it has a
 * consequence that reaches other devices, and half-finishing it is worse than
 * not starting.
 *
 * That consequence is stated on the form, before the button, rather than
 * reported afterwards. The server ends EVERY session when the password changes,
 * this browser's included, and immediately issues this browser a new one — so
 * the owner stays exactly where he is and every other device is signed out. He
 * should know that is about to happen while he can still decide not to.
 */

const MIN_PASSWORD = 12;

export function ChangePassword({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const change = useChangePassword();
  const busy = change.isPending;

  const tooShort = next.length > 0 && next.length < MIN_PASSWORD;
  const same = next !== "" && next === current;
  const complete = current !== "" && next.length >= MIN_PASSWORD && !same;

  const close = () => {
    onOpenChange(false);
    // Not left behind for the next time it opens: one of these is the password
    // that still works, and it should not be sitting in a field in a tab left
    // open on a desk.
    setCurrent("");
    setNext("");
    change.reset();
  };

  const edit = (set: (v: string) => void) => (v: string) => {
    set(v);
    if (change.isError) change.reset();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (busy) return; // mid-request: the outcome is decided, let it land
        if (o) onOpenChange(true);
        else close();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change password</DialogTitle>
          <DialogDescription>
            Every other device signs out. You stay signed in here.
          </DialogDescription>
        </DialogHeader>

        <form
          id="change-password"
          onSubmit={(e) => {
            e.preventDefault();
            if (busy || !complete) return;
            change.mutate(
              { current, next },
              {
                onSuccess: () => {
                  close();
                  // Nothing on screen changes, so without this the owner has no
                  // way to tell it worked.
                  toast.success("Password changed", {
                    description: "Every other device has been signed out.",
                  });
                },
              },
            );
          }}
          className="space-y-4"
        >
          <Field
            id="current-password"
            label="Current password"
            hint="Asked for even though you are signed in — a session is what somebody would have if they walked up to an unlocked laptop, which is the case this check is for."
            type="password"
            value={current}
            onChange={(e) => edit(setCurrent)(e.target.value)}
            disabled={busy}
            autoComplete="current-password"
            invalid={change.isError}
            describedBy={change.isError ? "change-error" : undefined}
          />

          <Field
            id="next-password"
            label="New password"
            hint={`At least ${MIN_PASSWORD} characters.`}
            type="password"
            value={next}
            onChange={(e) => edit(setNext)(e.target.value)}
            disabled={busy}
            autoComplete="new-password"
            invalid={tooShort || same}
          />

          {same && (
            <FormError
              id="change-same"
              message="That is the password you already have."
            />
          )}
          {change.isError && (
            <FormError id="change-error" message={change.error.message} />
          )}
        </form>

        <DialogFooter>
          <Button variant="outline" onClick={close} disabled={busy}>
            Cancel
          </Button>
          <Button type="submit" form="change-password" disabled={busy || !complete}>
            {busy ? (
              <>
                <Loader2 className="size-4 animate-spin" aria-hidden />
                Changing
              </>
            ) : (
              "Change password"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
