"use client";

import { useLayoutEffect, useRef } from "react";
import { ArrowUp, Check, ChevronDown, Info, Paperclip, Plus, Square, TriangleAlert, Undo2 } from "lucide-react";
import { cn } from "cn";
import type { Model, PendingSwitch, Provider, ProviderId } from "@/lib/chat-types";
import {
  formatBytes,
  MAX_FILE_BYTES,
  unreadableBy,
  useAttachments,
  useConversationAttachments,
  useRefusedFiles,
} from "@/lib/files";
import { providerStyle, formatTokens } from "./provider-meta";
import { AttachmentChip } from "./file-bits";
import { ProviderIcon } from "./provider-icon";
import { Button } from "@/components/ui/button";
import { Status } from "@/components/ui/status";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type Props = {
  /** Whose message box this is: the files waiting in it belong to the
   *  conversation, like the draft text. */
  conversationId: string | null;
  providers: Provider[];
  activeProvider: ProviderId;
  activeModel: Model;
  pending: PendingSwitch | null;
  disabled: boolean;
  disabledReason?: string;
  /** Amber by default. `neutral` for a state that is not a fault — see below. */
  disabledTone?: "warning" | "neutral";
  value: string;
  onChange: (v: string) => void;
  onSelect: (provider: ProviderId, model: Model) => void;
  onCancelSwitch: () => void;
  onSend: () => void;
  /** A turn is running and can be called back. The send button becomes the stop
   *  button rather than a second control appearing elsewhere: it is where the
   *  hand already is, and the working indicator it would otherwise sit beside
   *  scrolls out of sight on a long answer. */
  running: boolean;
  onStop: () => void;
};

export function Composer({
  conversationId,
  providers,
  activeProvider,
  activeModel,
  pending,
  disabled,
  disabledReason,
  disabledTone = "warning",
  value,
  onChange,
  onSelect,
  onCancelSwitch,
  onSend,
  running,
  onStop,
}: Props) {
  const taRef = useRef<HTMLTextAreaElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const attachments = useConversationAttachments(conversationId);
  const refused = useRefusedFiles(conversationId);
  const addFiles = useAttachments((s) => s.add);
  const removeFile = useAttachments((s) => s.remove);
  const retryFile = useAttachments((s) => s.retry);
  const uploading = attachments.some((a) => a.status === "uploading");
  const failed = attachments.some((a) => a.status === "failed");
  const canSend = !disabled && !uploading && !failed && (value.trim() !== "" || attachments.length > 0);
  const unreadable = unreadableBy(pending?.to ?? activeProvider, attachments);
  const add = (files: File[]) => {
    if (conversationId && files.length > 0) addFiles(conversationId, files);
  };

  // Fit the box to its text on every change, not only on typing, so it also
  // shrinks back when a send clears it or another conversation's draft loads.
  // It grows to 40% of the window (max-h), then scrolls; the old 200px cap
  // scrolled a pasted paragraph after eight lines.
  useLayoutEffect(() => {
    const el = taRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, [value]);

  const shown = pending
    ? { provider: pending.to, model: pending.toModel }
    : { provider: activeProvider, model: activeModel };
  const c = providerStyle(shown.provider);

  // The list is empty until the daemon reports what is installed, and on a
  // first load that is a couple of seconds during which this still renders.
  // Reaching into providers[0] there threw and took the whole page down.
  const shownProvider = providers.find((p) => p.id === shown.provider) ??
    providers[0] ?? { id: shown.provider, label: shown.provider, models: [] };

  return (
    <div className="shrink-0 border-t bg-card/40">
      {/* A switch costs nothing until the next message is sent, so the price is
          quoted here rather than charged on selection. Switching back before
          sending withdraws it entirely.

          Only when there is a price. A switch in an empty conversation, or to
          another model of the same agent, replays nothing — and the banner used
          to announce it anyway, as "codex hasn't seen this conversation. It
          will catch up on 0 messages" (docs/Bugs.md B-50). The menu already
          shows what was picked; choosing again undoes it. */}
      {pending && pending.messagesToReplay > 0 && (
        <div className="mx-auto flex max-w-3xl items-center gap-3 px-4 pt-3">
          <div className="flex flex-1 items-center gap-2.5 rounded-lg border border-dashed px-3.5 py-2 text-sm">
            <ProviderIcon
              provider={pending.to}
              className={`size-3.5 shrink-0 ${c.text}`}
            />
            <span className="flex-1 text-muted-foreground">
              <span className={c.text}>{pending.to}</span> hasn&apos;t seen this
              conversation. It will catch up on{" "}
              <span className="text-foreground">
                {pending.messagesToReplay} messages
              </span>{" "}
              (~
              <span className="font-mono text-foreground">
                {formatTokens(pending.estimatedTokens)} tokens
              </span>
              ) when you send.
            </span>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 shrink-0"
              onClick={onCancelSwitch}
            >
              <Undo2 className="size-3.5" />
              Keep {activeProvider}
            </Button>
          </div>
        </div>
      )}

      {disabled && disabledReason && (
        <div className="mx-auto max-w-3xl px-4 pt-3">
          {/* Nothing new can be sent until a person or the computer changes:
              worth an amber notice, and not a fault, so not red.

              Unless nothing is connected at all. Then it is not a notice about
              something going wrong, it is a step not taken yet (docs/Bugs.md
              B-46) — so it loses the colour that means "attention needed" and
              keeps only the sentence and the place to go. */}
          <p
            className={cn(
              "rounded-lg border px-3.5 py-2 text-sm",
              disabledTone === "warning"
                ? "border-warning/30 bg-warning-fill/50"
                : "bg-muted/55",
            )}
          >
            <Status tone={disabledTone} quiet>
              {disabledReason}
            </Status>
          </p>
        </div>
      )}

      <div className="mx-auto max-w-3xl px-4 py-3">
        {/* The text takes the whole width, with agent, model and send in one row
            under it (D-055). Beside the text, the pickers left a long message
            wrapping in two thirds of the box over an empty column. */}
        <div className="rounded-2xl border bg-background p-2 focus-within:ring-2 focus-within:ring-ring">
          {attachments.length > 0 && conversationId && (
            <div className="flex flex-wrap gap-1.5 px-0.5 pt-0.5 pb-1.5">
              {attachments.map((a) => (
                <AttachmentChip
                  key={a.key}
                  attachment={a}
                  onRemove={() => removeFile(conversationId, a.key)}
                  onRetry={() => retryFile(conversationId, a.key)}
                />
              ))}
            </div>
          )}
          <textarea
            ref={taRef}
            rows={1}
            value={value}
            disabled={disabled}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                if (canSend) onSend();
              }
            }}
            // A pasted screenshot arrives as a file. Pasted text is left alone.
            onPaste={(e) => {
              const files = Array.from(e.clipboardData.files);
              if (files.length > 0 && !disabled) {
                e.preventDefault();
                add(files);
              }
            }}
            placeholder={
              running
                ? `${shown.provider} is working — stop it to type`
                : disabled
                  ? "Unavailable"
                  : `Message ${shown.provider}…`
            }
            aria-label="Message"
            className="block max-h-[40vh] min-h-9 w-full resize-none bg-transparent px-1.5 py-2 text-[15px] outline-none placeholder:text-muted-foreground disabled:opacity-50"
          />

          <div className="flex items-center gap-2">
            {/* A menu rather than a straight file dialog, because this is where
                more ways of bringing something into a message will go (the
                owner's reference had sources and commands; later). */}
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon"
                    disabled={disabled || !conversationId}
                    aria-label="Add photos and files"
                    title="Add photos and files"
                    className="size-9 shrink-0 rounded-xl text-muted-foreground"
                  />
                }
              >
                <Plus className="size-[18px]" />
              </DropdownMenuTrigger>
              <DropdownMenuContent side="top" align="start" className="w-80">
                <DropdownMenuItem onClick={() => fileInput.current?.click()}>
                  <Paperclip className="size-4" />
                  <span className="font-medium whitespace-nowrap">Add photos &amp; files</span>
                  <span className="truncate text-muted-foreground">Upload from your computer</span>
                </DropdownMenuItem>
                <p className="mt-1 border-t px-2 pt-1.5 pb-1 text-xs text-muted-foreground">
                  Or drop them on the conversation, or paste a screenshot. Up to{" "}
                  {formatBytes(MAX_FILE_BYTES)} each.
                </p>
              </DropdownMenuContent>
            </DropdownMenu>
            <input
              ref={fileInput}
              type="file"
              multiple
              hidden
              onChange={(e) => {
                add(Array.from(e.target.files ?? []));
                // So choosing the same file again still counts as a change.
                e.target.value = "";
              }}
            />
            {/* Provider first, then that provider's models. One merged menu was
                tolerable at two models each; agy alone offers fourteen. */}
            {/* Allowed to shrink, and the model name truncates first, so a narrow
                pane still has room for the send button (docs/Bugs.md B-51). */}
            <div className="flex min-w-0 shrink items-center gap-0.5">
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={disabled}
                      aria-label={`Provider: ${shown.provider}`}
                      className="h-9 shrink-0 gap-1.5 rounded-xl px-2.5"
                    />
                  }
                >
                  <ProviderIcon
                    provider={shown.provider}
                    className={`size-4 ${c.text}`}
                  />
                  <span className="text-sm">{shown.provider}</span>
                  <ChevronDown className="size-3.5 text-muted-foreground" />
                </DropdownMenuTrigger>
  
                {/* Anchored to the bottom of the window, so it opens upward.
                    Never set a max-height here — the base component already caps
                    it at --available-height, and overriding that is what makes a
                    long menu run off the bottom of the screen. */}
                <DropdownMenuContent side="top" align="start" className="w-56">
                  {providers.map((p) => {
                    const blocked = p.availability === "blocked";
                    const pc = providerStyle(p.id);
                    return (
                      <DropdownMenuItem
                        key={p.id}
                        disabled={blocked}
                        onClick={() => onSelect(p.id, p.model ?? p.models[0])}
                      >
                        <ProviderIcon
                          provider={p.id}
                          className={`size-4 ${blocked ? "text-muted-foreground" : pc.text}`}
                        />
                        <span className="flex-1">{p.label}</span>
                        {p.id === shown.provider ? (
                          <Check className="size-4" />
                        ) : blocked ? (
                          <span className="text-xs text-muted-foreground">
                            {p.unavailableReason}
                          </span>
                        ) : null}
                      </DropdownMenuItem>
                    );
                  })}
                </DropdownMenuContent>
              </DropdownMenu>
  
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={disabled || shownProvider.models.length === 0}
                      aria-label={`Model: ${shown.model.label}`}
                      className="h-9 min-w-0 max-w-52 shrink gap-1.5 rounded-xl px-2.5 text-muted-foreground"
                    />
                  }
                >
                  <span className="truncate text-xs">{shown.model.label}</span>
                  <ChevronDown className="size-3.5 shrink-0" />
                </DropdownMenuTrigger>
  
                <DropdownMenuContent side="top" align="start" className="w-64">
                  {shownProvider.models.map((m) => (
                    <DropdownMenuItem
                      key={m.id}
                      onClick={() => onSelect(shownProvider.id, m)}
                    >
                      {m.id === shown.model.id ? (
                        <Check className="size-4" />
                      ) : (
                        <span className="size-4" aria-hidden />
                      )}
                      <span className="flex-1">{m.label}</span>
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
  
            {running ? (
              <Button
                size="icon"
                variant="secondary"
                className="ml-auto size-9 shrink-0 rounded-xl"
                onClick={onStop}
                aria-label="Stop this turn"
                title="Stop this turn"
              >
                {/* A filled square, the one stop glyph nobody has to learn. */}
                <Square className="size-3.5 fill-current" />
              </Button>
            ) : (
              <Button
                size="icon"
                className="ml-auto size-9 shrink-0 rounded-xl"
                disabled={!canSend}
                onClick={onSend}
                aria-label="Send message"
              >
                <ArrowUp className="size-4" />
              </Button>
            )}
          </div>
        </div>
        {(refused.length > 0 || unreadable.length > 0 || failed) && (
          <div className="mt-2 space-y-1 px-1 text-xs leading-4">
            {refused.map((r) => (
              <p key={r.name} className="flex items-start gap-1.5 text-destructive-text">
                <TriangleAlert className="mt-px size-3.5 shrink-0" aria-hidden />
                {r.name} is {formatBytes(r.size)}, over the {formatBytes(MAX_FILE_BYTES)} limit, so
                it was not added.
              </p>
            ))}
            {unreadable.length > 0 && (
              <p className="flex items-start gap-1.5 text-warning-text">
                <Info className="mt-px size-3.5 shrink-0" aria-hidden />
                {shown.provider} sees pictures and small text files. It can’t open{" "}
                {unreadable.join(", ")} with what it’s allowed to do today; agy or claude can.
              </p>
            )}
            {failed && (
              <p className="flex items-start gap-1.5 text-destructive-text">
                <TriangleAlert className="mt-px size-3.5 shrink-0" aria-hidden />
                A file didn’t upload. Try again or remove it to send.
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
