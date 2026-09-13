# DESIGN.md — sparstrowgen

> The design doctrine for this project. Every frontend agent reads this before designing or
> building anything. When this document and any other design guidance disagree, this wins.

| | |
|---|---|
| **Decided** | 2026-09-13, codified from the owner's chosen Chat direction and later account-access decisions |
| **Situation** | codify |
| **Component library** | shadcn/ui 4.21, Base UI, `base-nova`; Lucide icons |
| **References** | Claude Code desktop for its familiar two-pane working shell; the rejected terminal direction only for a provider-switch event inside the transcript; the rejected document direction only for the centred agent reading column |
| **Status** | agreed foundation; feature-specific layouts still require rendered owner choice |

## 1. North star

sparstrowgen is a conversation people leave open while doing real work. On a desktop monitor in a
focused working session, the interface should recede: the current conversation is easy to read,
the next action is predictable, and provider or computer constraints are visible without becoming
a dashboard. Familiarity is deliberate because the product already asks people to move among
several agents.

**Key characteristics**

- Dense enough for daily work, with more space around agent output than navigation rows.
- Flat layered surfaces separated by lightness and one-pixel borders, not decorative shadows.
- Restrained color. Provider identity and actionable state earn color; inactive chrome does not.
- One sans family for product UI and one mono family only for code, paths, tokens, time, and money.

## 2. Colour

All product colors are OKLCH-backed CSS variables in `apps/web/app/globals.css`. Components consume
semantic tokens only. A literal color in a component is a defect because it breaks every expression
except the one being viewed.

| Role | Light | Dark | Used for |
|---|---|---|---|
| Background | `--background: oklch(1 0 0)` | `oklch(0.155 0.006 260)` | Primary canvas |
| Card / raised surface | `--card: oklch(1 0 0)` | `oklch(0.195 0.007 260)` | Menus, composer layer, bounded controls |
| Foreground | `--foreground: oklch(0.145 0 0)` | `oklch(0.95 0.003 260)` | Primary text |
| Muted foreground | `--muted-foreground: oklch(0.556 0 0)` | `oklch(0.68 0.01 260)` | Metadata and secondary copy |
| Border | `--border: oklch(0.922 0 0)` | `oklch(1 0 0 / 10%)` | Structural separation |
| Primary action | `--primary: oklch(0.205 0 0)` | `oklch(0.92 0.003 260)` | The one primary action in a local decision |
| User message | `--mine: oklch(0.93 0.02 255)` | `oklch(0.3 0.028 258)` | Compact user prompts only |
| Destructive | `--destructive: oklch(0.577 0.245 27.325)` | `oklch(0.68 0.19 22)` | Errors and destructive actions |
| Claude identity | `--provider-claude: oklch(0.7 0.128 45)` | `oklch(0.74 0.13 45)` | Claude mark and attribution |
| Codex identity | `--provider-codex: oklch(0.82 0.012 250)` | `oklch(0.86 0.01 250)` | Codex mark and attribution |
| agy identity | `--provider-agy: oklch(0.7 0.115 255)` | `oklch(0.74 0.12 255)` | agy mark and attribution |

**Named rule: restrained color.** Saturated color occupies no more than 10% of a normal product
screen. Provider colors identify providers only. Destructive color identifies a fault or destructive
choice only. A status is never communicated by color alone.

## 3. Typography

- Product family: Geist Sans through `--font-geist-sans`.
- Literal family: Geist Mono through `--font-geist-mono`.
- Page or product title: 18px, medium, tight tracking.
- Surface title and primary row: 14px, medium.
- Body and controls: 14 to 15px, regular; form inputs use at least 16px at phone width.
- Metadata: 12px, regular; use tabular numerals for elapsed time, tokens, and money.
- Reading measure: agent prose never exceeds 48rem or 75 characters per line.

**Named rule: literal mono only.** Monospace is used only where character alignment or verbatim
content matters: code, raw transcript, paths, token counts, spend, and elapsed time.

## 4. Spacing & layout

Use a 4px base unit through the existing Tailwind spacing scale. Grouped controls use 4 to 12px
internal gaps; distinct regions use 24 to 32px separation. Desktop Chat keeps a persistent
conversation list and a flexible transcript. Agent output sits in a centred reading column no wider
than 48rem. User prompts align right and stay compact. At narrow widths the structure must collapse,
not merely squeeze; the exact mobile Chat pattern remains undecided under G-21.

## 5. Elevation & depth

Depth comes first from surface lightness and one-pixel borders. Shadows are limited to floating
menus, tooltips, dialogs, and a selected control lifted from a segmented track. Conversation rows,
messages, empty states, banners, and setup steps do not receive decorative shadows. The base radius
is `--radius: 0.625rem`; controls derive small, medium, large, and extra-large radii from that token.

## 6. Iconography

- **Set:** Lucide, with vendor marks kept as their real single-color SVG paths.
- **Sizes:** 12 to 14px for inline metadata, 16px for controls, 20 to 32px only for a genuine empty
  or error state.
- **Stroke / weight:** Lucide default stroke. Do not mix icon families.
- **Colour:** foreground or muted foreground by default; provider and destructive tokens only when
  the icon carries that exact meaning.

| Concept | Icon | Notes |
|---|---|---|
| New conversation | `Plus` | Paired with text except in a labelled icon button |
| Search | `Search` | Conversation search only |
| Folder / working directory | `FolderOpen` | The path is always visible beside it |
| Machine disconnected | `PlugZap` | Pair with explanatory text and the next action |
| Retry / reconnect | `RefreshCw` | Action, never ambient animation |
| Computer | `MonitorSmartphone` | Local component or paired computer |
| Destructive removal | `Trash2` | Disconnect uses its own wording; do not imply data deletion |
| Provider switch | `ArrowRightLeft` | Appears in the transcript event |
| Send / stop | `ArrowUp` / filled `Square` | Occupy the same composer position |

**Rule:** Icons clarify a named action, entity, or state. Decorative icon tiles and emoji stand-ins
are not permitted. Icon-only controls require an accessible name and tooltip where the meaning is
not universal.

## 7. Motion

| Movement | Duration | Easing | Applies to |
|---|---|---|---|
| Hover and focus response | 150ms | ease-out | Color and opacity feedback |
| Menu or panel reveal | 180ms | ease-out-quart | Floating or progressive surfaces |
| State replacement | 200ms | ease-out-quint | Setup and connection state changes |
| Active work indication | Until resolved | linear or discrete | A bounded indicator that stops when work stops |

Do not animate layout dimensions. No bounce, decorative entrances, or permanently pulsing status
dots. Under `prefers-reduced-motion`, remove nonessential transitions and retain state changes
without animation.

## 8. Component vocabulary

Use the installed shadcn/Base UI primitives before authoring a new primitive. `Button`, `Input`,
`DropdownMenu`, `AlertDialog`, `ScrollArea`, `Skeleton`, `Tooltip`, and Sonner are the current
vocabulary. A destructive action uses `AlertDialog`; a transient successful result may use a toast;
a condition requiring action remains inline. Long setup work is a page or progressive inline flow,
not a scrolling modal.

**Rule:** Business meaning stays outside `components/ui`. A missing primitive is checked against
the shadcn registry before being hand-built.

## 9. Expected surfaces & elements

| Element | Status | Where | Note |
|---|---|---|---|
| Conversation list plus reading pane | Expected | Chat | The familiar daily shell |
| Provider status strip | Expected | Chat | Supporting information, never a dashboard |
| Centred agent reading column | Expected | Chat | Agent output is the primary reading surface |
| Right-aligned compact user prompts | Expected | Chat | Prompts are context, not equal chat bubbles |
| Single-column account access | Expected | Account routes | One clear action per screen |
| Machines navigation | Expected | Persistent product sidebar | A dedicated destination because computer capabilities will expand |
| Machines list | Expected | Machines route | Shows only the signed-in person's computers and opens a focused profile |
| Machine profile | Expected | Separate machine route | Durable home for connection, providers, folders, disconnect and later approved capabilities |
| Progressive computer setup | Expected | Machines and first use | Setup begins from the dedicated destination and can be resumed later |
| Inline persistent machine attention state | Expected | Where Chat needs the machine | Must contain a next action |
| Disconnect confirmation | Expected | US2 computer action | Names the effect on active work and reconnection |
| Generic dashboard | Not building | — | Conversation remains the product's centre |
| Generic Settings page | Not building in Phase 1 | — | Only approved setup and update jobs are designed |
| Command palette | Not building | — | No proven navigation volume or owner request |
| Appearance preferences | Not yet | — | Separate draft feature after the Phase 1 gate |

The persistent product sidebar contains Chat and Machines. Chat remains the daily work surface;
Machines owns a route-based computer list, focused computer profiles, pairing and disconnect. A
machine profile may gain later approved capabilities without turning Chat into a management screen.

**Rule:** A new persistent surface or navigation mechanism is added here with owner approval before
it appears in an individual feature.

## 10. The four states

- **Populated:** Show real-length names, paths, provider labels, and the action that matters now.
- **Empty:** Explain what becomes possible and provide one direct next action. Never show placeholder
  charts, disabled furniture, or a dead end.
- **Loading:** Preserve the surface geometry with skeletons or bounded progress copy. Do not replace
  the whole product with a central spinner.
- **Error:** Say what failed, what remains safe, and the next valid action. Keep readable work visible
  when only sending or pairing is unavailable.

## 11. Named rules

1. Every component color resolves through a semantic token; no literal colors in components.
2. Saturated color occupies no more than 10% of a normal product screen.
3. Provider colors identify providers only; status is never color alone.
4. Monospace is reserved for literal or aligned content.
5. Agent prose is centred and no wider than 48rem; user prompts are compact and right-aligned.
6. Depth uses surface lightness and one-pixel borders; shadows are limited to floating surfaces and
   selected segmented controls.
7. Icons come from Lucide or the real provider marks; decorative icons and emoji are forbidden.
8. Motion conveys a state transition and respects reduced motion; layout dimensions do not animate.
9. All surfaces include populated, empty, loading, and error states before owner confirmation.
10. A new persistent surface or navigation mechanism requires owner approval in this document.

## 12. Do / Don't

**Do**

- Keep the current action close to its consequence and recovery.
- Use realistic product names, machine names, paths, timings, and errors in every design.
- Let useful ideas move between rejected and chosen directions.
- Preserve provider differences instead of forcing false visual symmetry.

**Don't**

- Do not turn provider or computer status into a cockpit of cards and gauges.
- Do not use nested cards, side accent stripes, gradient text, decorative glass, or icon tiles.
- Do not claim that an unanswered local launch proves the component is not installed.
- Do not hide destructive effects or token cost behind an action that already happened.
- Do not copy design rules into skills or component comments; point back to this document.

## 13. Deliberately undecided

- The final narrow-screen Chat structure, tracked as G-21.
- Semantic success, warning, and information tokens. US2 may communicate connection states with
  icon, label, foreground, muted, and destructive tokens until an approved palette adds them.
- The later account-wide appearance preferences and their Paper, Slate, Soft, and Mono expressions.
