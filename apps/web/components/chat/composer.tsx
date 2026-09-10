"use client";

import { useRef } from "react";
import { ArrowUp, Check, ChevronDown, Undo2 } from "lucide-react";
import type { PendingSwitch, Provider, ProviderId } from "@/lib/chat-types";
import { providerClasses, formatTokens } from "./provider-meta";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type Props = {
  providers: Provider[];
  activeProvider: ProviderId;
  activeModel: string;
  pending: PendingSwitch | null;
  disabled: boolean;
  disabledReason?: string;
  value: string;
  onChange: (v: string) => void;
  onSelect: (provider: ProviderId, model: string) => void;
  onCancelSwitch: () => void;
  onSend: () => void;
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
}: Props) {
  const taRef = useRef<HTMLTextAreaElement>(null);
  const shown = pending
    ? { provider: pending.to, model: pending.toModel }
    : { provider: activeProvider, model: activeModel };
  const c = providerClasses[shown.provider];

  return (
    <div className="shrink-0 border-t bg-card/40">
      {/* A switch costs nothing until the next message is sent, so the price is
          quoted here rather than charged on selection. Switching back before
          sending withdraws it entirely. */}
      {pending && (
        <div className="mx-auto flex max-w-3xl items-center gap-3 px-4 pt-3">
          <div className="flex flex-1 items-center gap-2.5 rounded-lg border border-dashed px-3.5 py-2 text-sm">
            <span className={`size-2 rounded-full ${c.dot}`} aria-hidden />
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
          <p className="rounded-lg border border-destructive/40 bg-destructive/10 px-3.5 py-2 text-sm text-muted-foreground">
            {disabledReason}
          </p>
        </div>
      )}

      <div className="mx-auto max-w-3xl px-4 py-3">
        <div className="flex items-end gap-2 rounded-2xl border bg-background p-2 focus-within:ring-2 focus-within:ring-ring">
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={disabled}
                  className="h-9 shrink-0 gap-1.5 rounded-xl px-2.5"
                />
              }
            >
              <span className={`size-2 rounded-full ${c.dot}`} aria-hidden />
              <span className="text-sm">{shown.provider}</span>
              <span className="text-xs text-muted-foreground">
                {shown.model}
              </span>
              <ChevronDown className="size-3.5 text-muted-foreground" />
            </DropdownMenuTrigger>

            <DropdownMenuContent align="start" className="w-64">
              {providers.map((p, i) => {
                const blocked = p.availability === "blocked";
                const pc = providerClasses[p.id];
                return (
                  <DropdownMenuGroup key={p.id}>
                    {i > 0 && <DropdownMenuSeparator />}
                    <DropdownMenuLabel className="flex items-center gap-2">
                      <span
                        className={`size-2 rounded-full ${pc.dot} ${blocked ? "opacity-50" : ""}`}
                        aria-hidden
                      />
                      <span className={blocked ? "text-muted-foreground" : ""}>
                        {p.label}
                      </span>
                      {blocked && (
                        <span className="ml-auto text-xs font-normal text-muted-foreground">
                          {p.unavailableReason}
                        </span>
                      )}
                    </DropdownMenuLabel>
                    {p.models.map((m) => {
                      const current =
                        shown.provider === p.id && shown.model === m;
                      return (
                        <DropdownMenuItem
                          key={m}
                          disabled={blocked}
                          onClick={() => onSelect(p.id, m)}
                          className="pl-7"
                        >
                          {current ? (
                            <Check className="size-4" />
                          ) : (
                            <span className="size-4" aria-hidden />
                          )}
                          {m}
                        </DropdownMenuItem>
                      );
                    })}
                  </DropdownMenuGroup>
                );
              })}
            </DropdownMenuContent>
          </DropdownMenu>

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
              disabled ? "Unavailable" : `Message ${shown.provider}…`
            }
            aria-label="Message"
            className="max-h-50 min-h-9 flex-1 resize-none bg-transparent py-2 text-[15px] outline-none placeholder:text-muted-foreground disabled:opacity-50"
          />

          <Button
            size="icon"
            className="size-9 shrink-0 rounded-xl"
            disabled={disabled || !value.trim()}
            onClick={onSend}
            aria-label="Send message"
          >
            <ArrowUp className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
