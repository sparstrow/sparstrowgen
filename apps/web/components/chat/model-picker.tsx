"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Check, ChevronDown, Info, Search, Star, TriangleAlert, X } from "lucide-react";
import { cn } from "cn";
import type { FavouriteModel, Model, Provider, ProviderId } from "@/lib/chat-types";
import { useFavouriteModels, useStarModel, useSwitchCost } from "@/lib/queries";
import { formatTokens, providerStyle } from "./provider-meta";
import { ProviderIcon } from "./provider-icon";
import { ProviderLimit } from "./provider-strip";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import { Status } from "@/components/ui/status";

/* One button for the agent and the model (the owner's reference, 2026-09-28;
   docs/design/prototypes/Chat/model-picker.handoff.md). Agents down the left,
   the chosen agent's models on the right, one search over every agent's models,
   and a starred list that holds models from any agent.

   A Popover rather than a DropdownMenu: it holds a text field, a tab rail and a
   list, and a menu's own roving focus fights the search box for the keys. */

type Props = {
  providers: Provider[];
  /** What the next message goes to: a pending switch, or the conversation's own. */
  shown: { provider: ProviderId; model: Model };
  /** The agent the conversation is on now. Moving away from it is what costs. */
  conversationProvider: ProviderId;
  conversationId: string | null;
  disabled: boolean;
  onSelect: (provider: ProviderId, model: Model) => void;
};

type View = "favourites" | string;
type Row = { provider: Provider; model: Model; key: string };

const FAVOURITES: View = "favourites";
const keyOf = (provider: string, model: string) => `${provider}\u0000${model}`;

export function ModelPicker({
  providers,
  shown,
  conversationProvider,
  conversationId,
  disabled,
  onSelect,
}: Props) {
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<View>(shown.provider);
  const [query, setQuery] = useState("");
  const [cursor, setCursor] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const favourites = useFavouriteModels();
  const star = useStarModel();
  const starred = useMemo(
    () => new Set((favourites.data ?? []).map((f) => keyOf(f.provider, f.model))),
    [favourites.data],
  );

  const c = providerStyle(shown.provider);
  const shownProvider = providers.find((p) => p.id === shown.provider);
  const usable = (p: Provider) => p.availability === "available";

  // What the list shows, as flat rows in screen order, so the keys and the
  // mouse move through the same sequence.
  const q = query.trim().toLowerCase();
  const rows: Row[] = useMemo(() => {
    const row = (provider: Provider, model: Model): Row => ({
      provider,
      model,
      key: keyOf(provider.id, model.id),
    });
    if (q) {
      return providers.filter(usable).flatMap((p) =>
        p.models
          .filter((m) => m.label.toLowerCase().includes(q) || m.id.toLowerCase().includes(q))
          .map((m) => row(p, m)),
      );
    }
    if (view === FAVOURITES) {
      // A starred model the computer no longer offers is hidden, not unstarred:
      // it comes back if the model does.
      return (favourites.data ?? []).flatMap((f: FavouriteModel) => {
        const p = providers.find((x) => x.id === f.provider);
        const m = p && usable(p) ? p.models.find((x) => x.id === f.model) : undefined;
        return p && m ? [row(p, m)] : [];
      });
    }
    const p = providers.find((x) => x.id === view);
    return p && usable(p) ? p.models.map((m) => row(p, m)) : [];
  }, [q, view, providers, favourites.data]);

  const active = Math.min(cursor, Math.max(rows.length - 1, 0));
  useEffect(() => {
    listRef.current
      ?.querySelector<HTMLElement>(`[data-row="${active}"]`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active, rows]);

  function openAt() {
    setView(shown.provider);
    setQuery("");
    const i = shownProvider?.models.findIndex((m) => m.id === shown.model.id) ?? -1;
    setCursor(Math.max(i, 0));
  }

  function choose(r: Row) {
    setOpen(false);
    onSelect(r.provider.id, r.model);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (rows.length === 0) return;
      const step = e.key === "ArrowDown" ? 1 : -1;
      setCursor((active + step + rows.length) % rows.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (rows[active]) choose(rows[active]);
    } else if (e.key === "Escape" && query) {
      // The first Esc clears the search; the next one closes.
      e.preventDefault();
      e.stopPropagation();
      setQuery("");
      setCursor(0);
    }
  }

  function toggleStar(r: Row) {
    star.mutate({
      favourite: { provider: r.provider.id, model: r.model.id },
      starred: !starred.has(r.key),
    });
  }

  // Before the computer has said what it has, there is nothing to pick from.
  if (providers.length === 0 && !disabled) {
    return (
      <span className="flex h-9 items-center gap-2 px-2.5" aria-label="Loading agents">
        <Skeleton className="size-4 rounded-full" />
        <Skeleton className="h-3 w-28" />
      </span>
    );
  }

  const listId = "model-picker-list";
  const shownLabel = shown.model.label || shown.model.id;

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        if (next) openAt();
        setOpen(next);
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            disabled={disabled}
            aria-label={`Agent and model: ${shown.provider}, ${shownLabel}`}
            className="group h-9 min-w-0 shrink gap-1.5 rounded-xl px-2.5"
          />
        }
      >
        <ProviderIcon provider={shown.provider} className={`size-4 shrink-0 ${c.text}`} />
        <span className="shrink-0 text-sm">{shown.provider}</span>
        {/* The model truncates first, so a narrow pane keeps room for send (B-51). */}
        <span className="min-w-0 truncate text-xs text-muted-foreground">{shownLabel}</span>
        <ChevronDown className="size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 group-aria-expanded:rotate-180 motion-reduce:transition-none" />
      </PopoverTrigger>

      {/* One fixed height, so moving between agents never makes it jump; less
          when the window is short. Opens upward from the composer. */}
      <PopoverContent
        side="top"
        align="start"
        sideOffset={6}
        initialFocus={searchRef}
        className="h-[min(440px,var(--available-height))] w-[min(440px,calc(100vw-16px))] gap-0 overflow-hidden p-0"
      >
        <div className="flex h-11 shrink-0 items-center gap-2 border-b pr-2 pl-3.5 text-muted-foreground">
          <Search className="size-4 shrink-0" aria-hidden />
          <input
            ref={searchRef}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setCursor(0);
            }}
            onKeyDown={onKeyDown}
            placeholder="Search models"
            aria-label="Search models"
            aria-controls={listId}
            aria-activedescendant={rows[active] ? `${listId}-${active}` : undefined}
            autoComplete="off"
            className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
          />
          {query && (
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Clear search"
              onClick={() => {
                setQuery("");
                setCursor(0);
                searchRef.current?.focus();
              }}
            >
              <X className="size-3.5" />
            </Button>
          )}
        </div>

        <div className="flex min-h-0 flex-1">
          <div
            role="tablist"
            aria-orientation="vertical"
            aria-label="Agents"
            className="flex w-13 shrink-0 flex-col items-center gap-0.5 border-r p-1.5"
          >
            <RailButton
              label="Favourites"
              selected={!q && view === FAVOURITES}
              onClick={() => {
                setView(FAVOURITES);
                setQuery("");
                setCursor(0);
                searchRef.current?.focus();
              }}
            >
              <Star className="size-[18px]" />
            </RailButton>
            <span className="my-1 h-px w-5 bg-border" aria-hidden />
            {providers.map((p) => {
              const off = !usable(p);
              return (
                <RailButton
                  key={p.id}
                  label={off ? `${p.label} · ${p.unavailableReason ?? "Unavailable"}` : p.label}
                  selected={!q && view === p.id}
                  onClick={() => {
                    setView(p.id);
                    setQuery("");
                    const i = p.id === shown.provider ? p.models.findIndex((m) => m.id === shown.model.id) : 0;
                    setCursor(Math.max(i, 0));
                    searchRef.current?.focus();
                  }}
                >
                  <ProviderIcon
                    provider={p.id}
                    className={cn("size-5", off ? "opacity-45" : providerStyle(p.id).text)}
                  />
                  {p.availability === "blocked" && (
                    <span className="absolute right-0.5 bottom-0.5 grid size-3.5 place-items-center rounded-full bg-popover">
                      <TriangleAlert className="size-2.5 text-warning" aria-hidden />
                    </span>
                  )}
                </RailButton>
              );
            })}
          </div>

          <div
            ref={listRef}
            id={listId}
            role="listbox"
            aria-label="Models"
            className="min-w-0 flex-1 overflow-y-auto overscroll-contain p-1.5"
          >
            {q ? (
              <SearchResults
                rows={rows}
                query={query}
                active={active}
                listId={listId}
                shown={shown}
                starred={starred}
                onHover={setCursor}
                onChoose={choose}
                onStar={toggleStar}
                onClear={() => {
                  setQuery("");
                  searchRef.current?.focus();
                }}
              />
            ) : view === FAVOURITES ? (
              <>
                <header className="px-2 pt-1.5 pb-2.5">
                  <p className="flex items-center gap-2 font-medium">
                    <Star className="size-4 fill-current" aria-hidden />
                    Favourites
                  </p>
                  <p className="mt-0.5 text-xs text-muted-foreground">Starred models from any agent</p>
                </header>
                {favourites.isError ? (
                  <p className="px-2 py-6 text-center text-sm text-muted-foreground">
                    Your starred models could not be loaded. The agents on the left still work.
                  </p>
                ) : favourites.isPending ? (
                  <div className="space-y-2 px-2">
                    <Skeleton className="h-4 w-40" />
                    <Skeleton className="h-4 w-32" />
                  </div>
                ) : rows.length === 0 ? (
                  <div className="flex flex-col items-center px-4 py-7 text-center text-sm text-muted-foreground">
                    <Star className="mb-2 size-6" aria-hidden />
                    <p className="font-medium text-foreground">No favourites yet</p>
                    <p>Star a model and it stays here, whichever agent it belongs to.</p>
                  </div>
                ) : (
                  rows.map((r, i) => (
                    <ModelRow
                      key={r.key}
                      row={r}
                      index={i}
                      listId={listId}
                      active={i === active}
                      current={r.provider.id === shown.provider && r.model.id === shown.model.id}
                      starred={starred.has(r.key)}
                      withMark
                      onHover={setCursor}
                      onChoose={choose}
                      onStar={toggleStar}
                    />
                  ))
                )}
              </>
            ) : (
              <ProviderPage
                provider={providers.find((p) => p.id === view)}
                rows={rows}
                active={active}
                listId={listId}
                shown={shown}
                starred={starred}
                conversationId={conversationId}
                quoteCost={view !== conversationProvider}
                onHover={setCursor}
                onChoose={choose}
                onStar={toggleStar}
              />
            )}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function RailButton({
  label,
  selected,
  onClick,
  children,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      aria-label={label}
      title={label}
      onClick={onClick}
      className="relative grid size-9.5 place-items-center rounded-md text-muted-foreground outline-none transition-colors duration-150 hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring aria-selected:bg-muted aria-selected:text-foreground aria-selected:ring-1 aria-selected:ring-border motion-reduce:transition-none"
    >
      {children}
    </button>
  );
}

type ListProps = {
  rows: Row[];
  active: number;
  listId: string;
  shown: { provider: ProviderId; model: Model };
  starred: Set<string>;
  onHover: (i: number) => void;
  onChoose: (r: Row) => void;
  onStar: (r: Row) => void;
};

function ProviderPage({
  provider,
  conversationId,
  quoteCost,
  ...list
}: ListProps & { provider: Provider | undefined; conversationId: string | null; quoteCost: boolean }) {
  const available = provider?.availability === "available";
  const cost = useSwitchCost(conversationId, provider?.id ?? "", quoteCost && available);
  if (!provider) return null;
  const pc = providerStyle(provider.id);

  // Grouped by maker only for an agent that routes to several (agy), and only
  // when there is more than one. The maker is the label's first word: derived,
  // not reported, so it only ever groups and never labels a model.
  const families = provider.routes ? [...new Set(list.rows.map((r) => r.model.label.split(" ")[0]))] : [];
  const grouped = families.length > 1;

  return (
    <>
      <header className="px-2 pt-1.5 pb-2.5">
        <p className="flex items-center gap-2 font-medium">
          <ProviderIcon provider={provider.id} className={`size-4 ${pc.text}`} />
          <span className={pc.text}>{provider.label}</span>
        </p>
        <div className="mt-0.5">
          {provider.availability === "blocked" ? (
            <Status tone="warning" className="text-xs">
              {provider.unavailableReason ?? "Unavailable"}
            </Status>
          ) : provider.availability === "waitable" ? (
            <span className="text-xs text-muted-foreground">{provider.unavailableReason}</span>
          ) : (
            <ProviderLimit provider={provider} />
          )}
        </div>
        {/* The price of switching, where the choice is made (D-055, B-50). */}
        {cost.data && cost.data.messagesToReplay > 0 && (
          <p className="mt-1.5 text-xs leading-4 text-muted-foreground">
            Hasn&apos;t seen this conversation. Catches up on{" "}
            <span className="text-foreground">{cost.data.messagesToReplay} messages</span> (~
            <span className="font-mono text-foreground">{formatTokens(cost.data.estimatedTokens)} tokens</span>)
            when you send.
          </p>
        )}
      </header>

      {!available ? (
        <p className="px-2 pb-3 text-sm text-muted-foreground">
          {provider.label} can&apos;t run on this computer right now. Once it can, its models appear
          here on their own.
        </p>
      ) : list.rows.length === 0 ? (
        <p className="px-2 pb-3 text-sm text-muted-foreground">
          {provider.label} hasn&apos;t said which models it offers yet.
        </p>
      ) : (
        list.rows.map((r, i) => {
          const family = r.model.label.split(" ")[0];
          const heading = grouped && (i === 0 || list.rows[i - 1].model.label.split(" ")[0] !== family);
          return (
            <div key={r.key}>
              {heading && (
                <p className={cn("px-2 pb-1 text-xs text-muted-foreground", i === 0 ? "pt-1" : "pt-2.5")}>
                  {family}
                </p>
              )}
              <ModelRow
                row={r}
                index={i}
                listId={list.listId}
                active={i === list.active}
                current={r.provider.id === list.shown.provider && r.model.id === list.shown.model.id}
                starred={list.starred.has(r.key)}
                onHover={list.onHover}
                onChoose={list.onChoose}
                onStar={list.onStar}
              />
            </div>
          );
        })
      )}

      {available && provider.routes && (
        <Footnote>
          {provider.label} runs models from several makers. Those from another agent&apos;s maker run
          through {provider.label} here, not through that agent.
        </Footnote>
      )}
      {/* codex is the one agent that cannot list its models (docs/KnownGaps.md
          G-8), so its list is ours and the plan behind it decides the rest. */}
      {available && provider.id === "codex" && (
        <Footnote>
          codex can&apos;t list its models, so these are the ones sparstrowgen knows. Your ChatGPT
          plan may not offer all of them.
        </Footnote>
      )}
    </>
  );
}

function SearchResults({
  query,
  onClear,
  ...list
}: ListProps & { query: string; onClear: () => void }) {
  if (list.rows.length === 0) {
    return (
      <div className="flex flex-col items-center px-4 py-7 text-center text-sm text-muted-foreground">
        <Search className="mb-2 size-6" aria-hidden />
        <p className="font-medium text-foreground">No model matches &ldquo;{query.trim()}&rdquo;</p>
        <p>Search looks at every agent&apos;s models.</p>
        <Button variant="outline" size="sm" className="mt-3" onClick={onClear}>
          Clear search
        </Button>
      </div>
    );
  }
  return list.rows.map((r, i) => {
    const heading = i === 0 || list.rows[i - 1].provider.id !== r.provider.id;
    const pc = providerStyle(r.provider.id);
    return (
      <div key={r.key}>
        {heading && (
          <p className={cn("flex items-center gap-1.5 px-2 pb-1 text-xs", i === 0 ? "pt-1" : "pt-2.5")}>
            <ProviderIcon provider={r.provider.id} className={`size-3 ${pc.text}`} />
            <span className={pc.text}>{r.provider.label}</span>
          </p>
        )}
        <ModelRow
          row={r}
          index={i}
          listId={list.listId}
          active={i === list.active}
          current={r.provider.id === list.shown.provider && r.model.id === list.shown.model.id}
          starred={list.starred.has(r.key)}
          highlight={query.trim()}
          onHover={list.onHover}
          onChoose={list.onChoose}
          onStar={list.onStar}
        />
      </div>
    );
  });
}

function ModelRow({
  row,
  index,
  listId,
  active,
  current,
  starred,
  withMark,
  highlight,
  onHover,
  onChoose,
  onStar,
}: {
  row: Row;
  index: number;
  listId: string;
  active: boolean;
  current: boolean;
  starred: boolean;
  withMark?: boolean;
  highlight?: string;
  onHover: (i: number) => void;
  onChoose: (r: Row) => void;
  onStar: (r: Row) => void;
}) {
  const label = row.model.label || row.model.id;
  return (
    <div
      id={`${listId}-${index}`}
      data-row={index}
      role="option"
      aria-selected={current}
      data-active={active || undefined}
      onMouseMove={() => !active && onHover(index)}
      onClick={() => onChoose(row)}
      className="group/row flex min-h-9 cursor-pointer items-center gap-2.5 rounded-md py-1.5 pr-1.5 pl-2 data-active:bg-accent"
    >
      <div className="min-w-0 flex-1">
        <p className={cn("flex min-w-0 items-center gap-1.5 text-sm", current && "font-medium")}>
          {withMark && (
            <ProviderIcon
              provider={row.provider.id}
              className={`size-3.5 shrink-0 ${providerStyle(row.provider.id).text}`}
            />
          )}
          <span className="truncate">
            <Highlighted text={label} match={highlight} />
          </span>
        </p>
        {row.model.description && (
          <p className="truncate text-xs text-muted-foreground" title={row.model.description}>
            {row.model.description}
          </p>
        )}
      </div>
      <button
        type="button"
        aria-pressed={starred}
        aria-label={`${starred ? "Unstar" : "Star"} ${label}`}
        title={starred ? "Remove from favourites" : "Add to favourites"}
        onClick={(e) => {
          e.stopPropagation();
          onStar(row);
        }}
        className={cn(
          "grid size-7 shrink-0 place-items-center rounded-md text-muted-foreground outline-none hover:text-foreground focus-visible:opacity-100 focus-visible:ring-2 focus-visible:ring-ring",
          starred ? "text-foreground" : "opacity-0 group-hover/row:opacity-100 group-data-active/row:opacity-100",
        )}
      >
        <Star className={cn("size-[15px]", starred && "fill-current")} />
      </button>
      <span className="grid size-4 shrink-0 place-items-center">
        {current && <Check className="size-4" aria-hidden />}
      </span>
    </div>
  );
}

function Highlighted({ text, match }: { text: string; match?: string }) {
  if (!match) return text;
  const i = text.toLowerCase().indexOf(match.toLowerCase());
  if (i < 0) return text;
  return (
    <>
      {text.slice(0, i)}
      <span className="underline underline-offset-2">{text.slice(i, i + match.length)}</span>
      {text.slice(i + match.length)}
    </>
  );
}

function Footnote({ children }: { children: React.ReactNode }) {
  return (
    <p className="mx-2 mt-2 mb-1 flex items-start gap-1.5 border-t pt-2 text-xs leading-4 text-muted-foreground">
      <Info className="mt-px size-3 shrink-0" aria-hidden />
      <span>{children}</span>
    </p>
  );
}
