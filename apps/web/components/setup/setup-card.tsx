"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { X } from "lucide-react";
import { useSetup } from "@/lib/queries";
import { useSetupView } from "@/lib/store";
import { Button } from "@/components/ui/button";
import { Status } from "@/components/ui/status";

/* The compact rendering of the same setup steps (D-046), for a setup that was
   skipped or interrupted. It reopens the wizard; it never carries its own copy
   of a step, so adding one changes `useSetup` and this follows. */

export function SetupCard() {
  const router = useRouter();
  const setup = useSetup();
  const { skipped, loaded, unskip, cardHidden, hideCard, loadSkipped } = useSetupView();

  // Reads the browser's flag itself, so the card is correct wherever it is
  // mounted rather than only on the page that happens to run the redirect.
  useEffect(() => loadSkipped(), [loadSkipped]);

  // Only when there is something outstanding, the answer is real rather than a
  // pending query (B-46), and setup was actually put aside — while the wizard
  // is the current screen this would be a second copy of it.
  if (!loaded || !skipped || cardHidden || !setup.settled || setup.complete) return null;

  return (
    <div className="relative mx-2 mt-2 rounded-lg border bg-background p-3">
      <Button
        variant="ghost"
        size="icon"
        className="absolute top-1.5 right-1.5 size-6"
        aria-label="Hide setup"
        onClick={() => hideCard()}
      >
        <X className="size-3.5" />
      </Button>
      <h3 className="text-[13px] font-semibold">Finish setting up</h3>
      <p className="mt-px text-xs text-muted-foreground">
        {setup.done} of {setup.total} done
      </p>
      <ul className="mt-2.5 flex flex-col gap-1.5">
        {setup.steps.map((step) => (
          <li key={step.id}>
            <Status tone={step.done ? "success" : "pending"} size="sm" className="text-[13px]">
              <span className={step.done ? "text-muted-foreground line-through" : undefined}>
                {step.task}
              </span>
            </Status>
          </li>
        ))}
      </ul>
      <Button
        className="mt-3 w-full"
        size="sm"
        onClick={() => {
          unskip();
          router.push("/setup");
        }}
      >
        {setup.steps.find((s) => !s.done)?.task ?? "Finish setting up"}
      </Button>
    </div>
  );
}
