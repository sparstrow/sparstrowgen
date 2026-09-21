"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Check, Download, MessageSquareText, MonitorSmartphone, RefreshCw } from "lucide-react";
import { cn } from "cn";
import { useMachines, useRealtime, useSetup, type SetupStep } from "@/lib/queries";
import { useSetupView } from "@/lib/store";
import { usePairing } from "@/lib/pairing";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ProfileFields } from "@/components/settings/profile-fields";
import { StatusIcon, type StatusTone } from "@/components/ui/status";

/* First-run setup: the full-screen rendering of the setup steps (D-046).
   No rail, no pane, no tray — there is nowhere else to be while it runs.

   Install is not a step and cannot be. The daemon dials out only, so nothing
   can ask this computer whether the component is present; opening the
   `sparstrowgen://` link IS the check, and install help is the branch taken
   when nothing answers. Ticking an "Install" box would assert something we have
   no way to know. */

function Stepper({ steps, currentId }: { steps: SetupStep[]; currentId: string | null }) {
  // One step is not a sequence, and a progress indicator with no progress in it
  // is decoration. It appears when there is something to be partway through —
  // which is when Profile and Workspace are built.
  if (steps.length < 2) return null;
  return (
    <ol className="mb-7 flex flex-wrap items-center justify-center gap-x-1.5 gap-y-2">
      {steps.map((step, i) => {
        const current = step.id === currentId;
        return (
          <li key={step.id} className="flex items-center gap-1.5">
            {i > 0 && <span className="h-px w-5 bg-border" aria-hidden />}
            <span
              className={cn(
                "flex items-center gap-2 text-sm whitespace-nowrap",
                current ? "font-medium text-foreground" : "text-muted-foreground",
              )}
              aria-current={current ? "step" : undefined}
            >
              <span
                className={cn(
                  "flex size-5.5 shrink-0 items-center justify-center rounded-full text-[11px] font-medium",
                  step.done
                    ? "bg-success text-background"
                    : current
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted text-muted-foreground",
                )}
              >
                {step.done ? <Check className="size-3" aria-hidden /> : i + 1}
              </span>
              {step.label}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

function Card({
  tone,
  icon,
  title,
  children,
  facts,
  actions,
}: {
  tone?: StatusTone;
  icon?: React.ReactNode;
  title: string;
  children: React.ReactNode;
  facts?: [string, string][];
  actions?: React.ReactNode;
}) {
  return (
    <section className="mt-6 rounded-lg border px-5 py-4">
      <div className="flex items-start gap-3">
        <span className="mt-0.5 shrink-0">
          {tone ? <StatusIcon tone={tone} size="md" /> : icon}
        </span>
        <div className="min-w-0">
          <h2 className="text-[15px] font-semibold">{title}</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">{children}</p>
        </div>
      </div>
      {facts && (
        <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 border-t pt-3.5 text-sm">
          {facts.map(([k, v]) => (
            <div key={k} className="contents">
              <dt className="text-muted-foreground">{k}</dt>
              <dd className="min-w-0 break-words">{v}</dd>
            </div>
          ))}
        </dl>
      )}
      {actions && <div className="mt-4 flex flex-wrap gap-2">{actions}</div>}
    </section>
  );
}

export function SetupWizard() {
  // The wizard replaces the shell, so it has to open the socket the shell would
  // have opened — nothing else on this route does. Without it the last screen
  // says "none reported yet" forever while the computer is sitting there online
  // with three agents, because nobody is listening for the event that says so.
  //
  // Still exactly one caller: `useRealtime` makes a socket per call, and this
  // route never mounts `AppShell`.
  useRealtime();

  const router = useRouter();
  const setup = useSetup();
  const machines = useMachines();
  const { skip, unskip } = useSetupView();
  const pair = usePairing();
  // The computer that was just connected, so the last screen can name it. Taken
  // from the list rather than kept from the approve call, because the list is
  // what every other surface reads.
  const [finished, setFinished] = useState(false);

  const connected = machines.data ?? [];
  const newest = connected[connected.length - 1];
  const ready = finished && !!newest;

  // A step passed over for now. Not persisted and not the same as skipping
  // setup: "I will name myself later, let me connect the computer" leaves the
  // step outstanding, so the resume card still lists it.
  const [passed, setPassed] = useState<Set<string>>(new Set());
  const steps = ready ? setup.steps.map((s) => ({ ...s, done: true })) : setup.steps;
  const current = ready ? null : (steps.find((s) => !s.done && !passed.has(s.id)) ?? null);
  const currentId = current?.id ?? null;
  const stepNumber = steps.findIndex((s) => s.id === currentId) + 1;
  // Every step either done or passed over, with nothing connected: there is no
  // step to show, so the last screen is the honest one.
  const nothingLeft = !ready && current === null;

  function leave(to: string) {
    unskip();
    router.push(to);
  }

  let body: React.ReactNode;
  if (!setup.settled && !machines.isError) {
    body = (
      <section className="mt-6 space-y-3 rounded-lg border px-5 py-5" aria-busy="true">
        <Skeleton className="h-4 w-2/5" />
        <Skeleton className="h-3 w-3/4" />
        <Skeleton className="h-8 w-44" />
      </section>
    );
  } else if (pair.stage === "error" || machines.isError) {
    body = (
      <Card
        tone="danger"
        title="Setup could not be started"
        actions={
          <Button onClick={() => pair.begin()}>
            <RefreshCw />
            Try again
          </Button>
        }
      >
        {pair.error ?? (machines.error as Error)?.message}. Your account is fine and nothing was
        connected.
      </Card>
    );
  } else if (currentId === "profile") {
    // The editor is the same component Settings → Account uses, so there is one
    // profile form in the app rather than two that drift. Saving a name makes
    // this step done, which moves the wizard on by itself — there is no
    // "current step" kept anywhere to get out of step with the account.
    body = (
      <section className="mt-6 rounded-lg border px-5 py-5">
        <ProfileFields />
        <p className="mt-5 border-t pt-4 text-sm text-muted-foreground">
          Only the name matters, and you can change it whenever you like.{" "}
          <button
            type="button"
            className="underline underline-offset-2 hover:text-foreground"
            onClick={() => setPassed((p) => new Set(p).add("profile"))}
          >
            Do this later
          </button>
        </p>
      </section>
    );
  } else if (ready) {
    body = (
      <Card
        tone="success"
        title={`${newest.name} is connected`}
        facts={[
          [
            "Agents found",
            newest.providers.length
              ? newest.providers.map((p) => p.label).join(", ")
              : "none reported yet",
          ],
          ["Version", newest.version || "unknown"],
        ]}
        actions={
          <>
            <Button onClick={() => leave("/")}>
              <MessageSquareText />
              Start your first conversation
            </Button>
            <Button variant="outline" onClick={() => leave(`/machines/${newest.id}`)}>
              See this computer
            </Button>
          </>
        }
      >
        It stays connected when you close this tab, and starts again when you sign in to Windows.
      </Card>
    );
  } else if (pair.stage === "approval") {
    body = (
      <Card
        tone="success"
        title={pair.machineName ? `Approve ${pair.machineName}?` : "Approve this computer?"}
        actions={
          <>
            <Button
              disabled={pair.approving}
              onClick={() => void pair.approve().then((ok) => ok && setFinished(true))}
            >
              {pair.approving ? "Connecting…" : "Approve computer"}
            </Button>
            <Button variant="outline" disabled={pair.approving} onClick={() => pair.notNow()}>
              Not now
            </Button>
          </>
        }
      >
        It will be able to run coding agents for your account. You can disconnect it at any time.
      </Card>
    );
  } else if (pair.stage === "waiting") {
    body = (
      <Card tone="progress" title="Looking for this computer">
        Approve the browser prompt to open sparstrowgen on this computer. This usually takes a second
        or two.
      </Card>
    );
  } else if (pair.stage === "unanswered") {
    // Never "it is not installed": an unanswered link cannot be told apart from
    // a slow start, a blocked handler, or a claim refused because another
    // account holds this computer (docs/KnownGaps.md G-40).
    body = (
      <Card
        tone="warning"
        title="This computer has not answered yet"
        actions={
          <>
            <Button onClick={() => pair.retry()}>
              <RefreshCw />
              Try again
            </Button>
            <Button variant="outline" nativeButton={false} render={<Link href="/install" />}>
              <Download />
              Install for Windows
            </Button>
          </>
        }
      >
        It may still be starting, or it may not be installed here. Nothing is broken either way — try
        again, or install it first.
      </Card>
    );
  } else if (nothingLeft) {
    // Either setup is genuinely finished — reached by hand, or completed in
    // another tab — or every remaining step was passed over just now. Offering
    // to connect under "Connect this computer" would read as though the last
    // attempt had not worked. Adding a SECOND computer is a real thing to want,
    // but Machines is where that lives; this route is first-run setup.
    const left = steps.filter((s) => !s.done);
    body = (
      <Card
        tone={left.length ? "info" : "success"}
        title={left.length ? "Nothing more for now" : "Setup is already finished"}
        actions={
          <>
            <Button onClick={() => leave("/")}>
              <MessageSquareText />
              Go to Chat
            </Button>
            <Button variant="outline" onClick={() => leave("/machines")}>
              {left.some((s) => s.id === "machines") ? "Connect a computer" : "Add another computer"}
            </Button>
          </>
        }
      >
        {left.length
          ? `You can still ${left.map((s) => s.task.toLowerCase()).join(" and ")} — it is waiting for you in Chat.`
          : "Everything here is done. Nothing needs connecting again."}
      </Card>
    );
  } else {
    body = (
      <Card
        icon={<MonitorSmartphone className="size-[18px]" aria-hidden />}
        title="Open sparstrowgen on this computer"
        actions={
          <>
            <Button onClick={() => pair.begin()}>Connect this computer</Button>
            <Button variant="outline" nativeButton={false} render={<Link href="/install" />}>
              <Download />I have not installed it yet
            </Button>
          </>
        }
      >
        Your browser will ask for permission to open it. There is no code to copy, and nothing
        connects in to your computer — it dials out to us.
      </Card>
    );
  }

  // The words belong to whichever step is showing, so the heading never
  // describes one thing while the card below it does another.
  const title =
    ready || nothingLeft
      ? {
          heading: "You’re ready",
          lede: ready
            ? "That is everything. Your computer is connected and its agents are available to every conversation."
            : "You can pick up anything left over from inside the app, whenever you want to.",
        }
      : currentId === "profile"
        ? {
            heading: "Your profile",
            lede: "A name and a picture, so sparstrowgen refers to you as you rather than as your email address.",
          }
        : {
            heading: "Connect this computer",
            lede: "sparstrowgen runs coding agents on a computer of your own. Connect the one you are sitting at, and its agents become available here.",
          };

  return (
    <main className="flex h-full flex-col overflow-y-auto">
      <div className="flex h-14 flex-none items-center justify-between border-b px-5">
        <span className="flex items-center gap-2.5 font-semibold tracking-tight">
          <span aria-hidden>s</span>sparstrowgen
        </span>
        {steps.length > 1 && !ready && (
          <span className="text-xs text-muted-foreground">
            Step {stepNumber} of {steps.length}
          </span>
        )}
      </div>

      <div className="mx-auto w-full max-w-xl px-6 pt-10 pb-12">
        <Stepper steps={steps} currentId={currentId} />
        <h1 className="text-center text-[22px] leading-7 font-semibold tracking-tight">
          {title.heading}
        </h1>
        <p className="mx-auto mt-2 max-w-md text-center text-sm text-muted-foreground">
          {title.lede}
        </p>

        {body}

        {!ready && !nothingLeft && (
          <div className="mt-5 flex items-center justify-between gap-3 border-t pt-4 text-sm text-muted-foreground">
            <span>You can finish this later. Your account is already usable.</span>
            <Button
              variant="outline"
              size="sm"
              className="shrink-0"
              onClick={() => {
                skip();
                router.push("/");
              }}
            >
              Skip for now
            </Button>
          </div>
        )}
      </div>

    </main>
  );
}
