/* How wide the resizable panes are: the shell's list pane and the files pane.

   A choice about one window, like the rail pin (docs/Decisions.md D-043), so it
   is remembered in this browser and never sent anywhere. The width lives in a
   CSS variable on <html> rather than in React state: a drag then changes one
   property per frame and re-renders nothing, and the script below can put the
   saved width in place before the first paint, so a reload does not open at
   the default and then jump. */

export type PaneId = "list" | "files";

type Pane = {
  cssVar: string;
  /** Used when nothing is saved, and by the CSS fallback before any script. */
  initial: number;
  min: number;
  max: number;
};

export const PANES: Record<PaneId, Pane> = {
  list: { cssVar: "--pane-list", initial: 212, min: 200, max: 440 },
  files: { cssVar: "--pane-files", initial: 400, min: 300, max: 860 },
};

const KEY = "sparstrowgen.pane-widths";

export function clampWidth(id: PaneId, px: number): number {
  const { min, max } = PANES[id];
  return Math.round(Math.min(max, Math.max(min, px)));
}

/** Moves the pane. Cheap enough to call on every pointer move. */
export function applyWidth(id: PaneId, px: number) {
  document.documentElement.style.setProperty(PANES[id].cssVar, `${clampWidth(id, px)}px`);
}

/** Remembers the width for the next visit. Storage can throw — a private
 *  window, or site data blocked — and the resize still holds for this one. */
export function saveWidth(id: PaneId, px: number) {
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) ?? "null") ?? {};
    localStorage.setItem(KEY, JSON.stringify({ ...saved, [id]: clampWidth(id, px) }));
  } catch {
    // Nothing to do: the width is already on screen.
  }
}

/** The width that shows every name in the pane without cutting it off, as
 *  Excel fits a column to its contents.
 *
 *  Every line of text in a pane ellipsises itself, so nothing on screen reports
 *  the width its content wants. A Range over a clipped element still measures
 *  its whole laid-out text, so the gap between that and the room the element
 *  has is how much wider (or, when negative, narrower) the pane could be. The
 *  pane follows the line that needs the most. Text marked data-autofit="skip"
 *  is left out — a full folder path would otherwise always win.
 *
 *  Only lines whose room changes with the pane's width can say what the pane
 *  needs. A heading as wide as its own text always reads as exactly full, and
 *  counting it would stop the pane ever narrowing. So the pane is widened by a
 *  probe for the length of one measurement, and a line counts by how much of
 *  that it was given; the width is put back before the browser paints, so the
 *  probe is never seen.
 *
 *  Returns null when the pane has no line that follows its width. */
export function fitWidth(pane: HTMLElement): number | null {
  const current = pane.getBoundingClientRect().width;
  const lines: { el: HTMLElement; room: number; text: number }[] = [];
  const range = document.createRange();
  for (const el of pane.querySelectorAll<HTMLElement>("*")) {
    if (el.clientWidth === 0) continue; // not displayed
    const style = getComputedStyle(el);
    if (style.textOverflow !== "ellipsis" || style.overflowX === "visible") continue;
    if (el.closest('[data-autofit="skip"]')) continue;
    range.selectNodeContents(el);
    const padding = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
    lines.push({ el, room: el.clientWidth, text: range.getBoundingClientRect().width + padding });
  }
  range.detach();

  const PROBE = 64;
  const { width, maxWidth } = pane.style;
  pane.style.width = `${current + PROBE}px`;
  pane.style.maxWidth = "none";
  const share = lines.map(({ el, room }) => (el.clientWidth - room) / PROBE);
  pane.style.width = width;
  pane.style.maxWidth = maxWidth;

  let need = -Infinity;
  lines.forEach(({ room, text }, i) => {
    // A line given under a quarter of the extra width is not following it.
    if (share[i] >= 0.25) need = Math.max(need, (text - room) / share[i]);
  });
  // One pixel over, because text measured at a fractional width can still be
  // given an ellipsis when it lands exactly on the edge.
  return need === -Infinity ? null : current + Math.ceil(need) + 1;
}

/** Runs before React in a <script> in the document head, beside the
 *  appearance's, so a saved width is the first one painted. Any failure leaves
 *  the defaults the CSS already falls back to. */
export const PANE_WIDTH_SCRIPT = `(function(){try{
var w=JSON.parse(localStorage.getItem(${JSON.stringify(KEY)})||"null")||{};
var p=${JSON.stringify(PANES)};
for(var k in p){var v=w[k];if(typeof v==="number"&&isFinite(v)){
document.documentElement.style.setProperty(p[k].cssVar,Math.min(p[k].max,Math.max(p[k].min,Math.round(v)))+"px");}}
}catch(e){}})();`;
