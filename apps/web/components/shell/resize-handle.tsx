"use client";

import { useEffect, useRef, useState } from "react";
import { cn } from "cn";
import { PANES, applyWidth, clampWidth, fitWidth, saveWidth, type PaneId } from "@/lib/pane-width";

/* The line on a pane's edge that resizes it: drag it, double-click it to fit
   the pane to its longest name, or focus it and use the arrow keys.

   Hand-built rather than shadcn's Resizable. That one (react-resizable-panels)
   owns the layout as a panel group, and both panes here change shape by CSS
   breakpoint alone — the list becomes the whole screen on a phone and the files
   pane floats over the conversation below lg — which a panel group could only
   follow through JS breakpoints. Multica's table columns resize the same way,
   with drag, double-click auto-fit and keys (docs/Decisions.md D-057).

   Sits inside the pane, which must be positioned, on the edge that faces the
   page. */

const STEP = 16;

export function ResizeHandle({
  pane,
  edge,
  label,
  onResizeStart,
  className,
}: {
  pane: PaneId;
  /** Which of the pane's edges this is. Dragging away from the pane widens it. */
  edge: "left" | "right";
  label: string;
  /** Before the width changes, for a pane with a size mode of its own to leave. */
  onResizeStart?: () => void;
  className?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);
  const { min, max } = PANES[pane];

  const panel = () => ref.current?.parentElement ?? null;

  // The width for a screen reader, written straight onto the element like the
  // width itself, so a drag still re-renders nothing.
  const report = () => {
    const el = panel();
    if (el) ref.current?.setAttribute("aria-valuenow", String(Math.round(el.getBoundingClientRect().width)));
  };
  useEffect(report);

  /** Starts from the width on screen, not the saved one: a pane held narrower
   *  by the window, or widened by its own mode, moves from where it is. */
  function begin(): number | null {
    const el = panel();
    if (!el) return null;
    const width = el.getBoundingClientRect().width;
    applyWidth(pane, width);
    onResizeStart?.();
    return width;
  }

  /** Moves the pane and returns the width it actually took. The page can hold
   *  a pane narrower than asked — the files pane never covers more than the
   *  conversation — and the width kept is the one on screen, so dragging back
   *  moves the edge at once instead of first unwinding width nobody saw. */
  function place(px: number): number {
    applyWidth(pane, px);
    const shown = panel()?.getBoundingClientRect().width ?? px;
    if (shown < clampWidth(pane, px) - 0.5) {
      applyWidth(pane, shown);
      return clampWidth(pane, shown);
    }
    return clampWidth(pane, px);
  }

  function set(px: number) {
    saveWidth(pane, place(px));
    report();
  }

  function onPointerDown(e: React.PointerEvent<HTMLDivElement>) {
    if (e.button !== 0) return;
    e.preventDefault();
    const start = begin();
    if (start === null) return;
    const handle = e.currentTarget;
    const startX = e.clientX;
    const sign = edge === "right" ? 1 : -1;
    let last = start;
    handle.setPointerCapture(e.pointerId);
    setDragging(true);
    // The pointer leaves the 8px handle as soon as it moves, so the cursor and
    // the no-text-selection go on the whole page for the length of the drag.
    const root = document.documentElement;
    root.style.cursor = "col-resize";
    root.style.userSelect = "none";

    const move = (ev: PointerEvent) => {
      last = place(start + sign * (ev.clientX - startX));
    };
    const end = () => {
      handle.removeEventListener("pointermove", move);
      handle.removeEventListener("pointerup", end);
      handle.removeEventListener("pointercancel", end);
      root.style.cursor = "";
      root.style.userSelect = "";
      setDragging(false);
      saveWidth(pane, last);
      report();
    };
    handle.addEventListener("pointermove", move);
    handle.addEventListener("pointerup", end);
    handle.addEventListener("pointercancel", end);
  }

  function autoFit() {
    const el = panel();
    if (!el || begin() === null) return;
    // Nothing to fit to — an empty list — goes back to where it started out.
    set(fitWidth(el) ?? PANES[pane].initial);
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
    const grow = edge === "right" ? "ArrowRight" : "ArrowLeft";
    const shrink = edge === "right" ? "ArrowLeft" : "ArrowRight";
    let next: number | null = null;
    if (e.key === grow || e.key === shrink) {
      const start = begin();
      if (start !== null) next = start + (e.key === grow ? STEP : -STEP);
    } else if (e.key === "Home") {
      next = min;
    } else if (e.key === "End") {
      next = max;
    } else if (e.key === "Enter") {
      e.preventDefault();
      autoFit();
      return;
    }
    if (next === null) return;
    e.preventDefault();
    if (e.key === "Home" || e.key === "End") begin();
    set(next);
  }

  return (
    <div
      ref={ref}
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-valuemin={min}
      aria-valuemax={max}
      tabIndex={0}
      title="Drag to resize · double-click to fit"
      onPointerDown={onPointerDown}
      onDoubleClick={autoFit}
      onKeyDown={onKeyDown}
      className={cn(
        // 8px to grab, centred on the pane's border; the 2px line inside it
        // shows only while it is being pointed at, dragged or focused.
        "group/resize absolute inset-y-0 z-30 w-2 cursor-col-resize touch-none outline-none select-none",
        edge === "right" ? "-right-1" : "-left-1",
        className,
      )}
    >
      <span
        aria-hidden
        className={cn(
          "absolute inset-y-0 left-1/2 w-0.5 -translate-x-1/2 bg-transparent transition-colors duration-100",
          "group-hover/resize:bg-ring/70 group-focus-visible/resize:bg-ring",
          dragging && "bg-ring transition-none",
        )}
      />
    </div>
  );
}
