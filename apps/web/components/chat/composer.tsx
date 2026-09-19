"use client";

import { useRef } from "react";
import { ArrowUp, Check, ChevronDown, Square, Undo2 } from "lucide-react";
import type { Model, PendingSwitch, Provider, ProviderId } from "@/lib/chat-types";
import { providerStyle, formatTokens } from "./provider-meta";
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
  providers: Provider[];
  activeProvider: ProviderId;
  activeModel: Model;
  pending: PendingSwitch | null;
  disabled: boolean;
  disabledReason?: string;
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
  providers,
  activeProvider,
  activeModel,
  pending,
  disabled,
  disabledReason,
  value,
  onChange,
  onSelect,
  onCancelSwitch,
  onSend,
  running,
  onStop,
}: Props) {
  const taRef = useRef<HTMLTextAreaElement>(null);
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
          sending withdraws it entirely. */}
      {pending && (
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
              worth an amber notice, and not a fault, so not red. */}
          <p className="rounded-lg border border-warning/30 bg-warning-fill/50 px-3.5 py-2 text-sm">
            <Status tone="warning" quiet>
              {disabledReason}
            </Status>
          </p>
        </div>
      )}

      <div className="mx-auto max-w-3xl px-4 py-3">
        <div className="flex items-end gap-2 rounded-2xl border bg-background p-2 focus-within:ring-2 focus-within:ring-ring">
          {/* Provider first, then that provider's models. One merged menu was
              tolerable at two models each; agy alone offers fourteen. */}
          <div className="flex shrink-0 items-center gap-0.5">
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={disabled}
                    aria-label={`Provider: ${shown.provider}`}
                    className="h-9 gap-1.5 rounded-xl px-2.5"
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
                    className="h-9 max-w-52 gap-1.5 rounded-xl px-2.5 text-muted-foreground"
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

          <textarea
            ref={taRef}
            rows={1}
            value={value}
            disabled={disabled}
            onChange={(e) => {
              onChange(e.target.value);
              const el = e.currentTarget;
              el.style.height = "auto";
              el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                if (value.trim() && !disabled) onSend();
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
            className="max-h-50 min-h-9 flex-1 resize-none bg-transparent py-2 text-[15px] outline-none placeholder:text-muted-foreground disabled:opacity-50"
          />

          {running ? (
            <Button
              size="icon"
              variant="secondary"
              className="size-9 shrink-0 rounded-xl"
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
              className="size-9 shrink-0 rounded-xl"
              disabled={disabled || !value.trim()}
              onClick={onSend}
              aria-label="Send message"
            >
              <ArrowUp className="size-4" />
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
