# Refuse list — absolute design slop

38 tells that read as machine-made in **any** app. Nothing here depends on this
project's doctrine; project-specific rules live in [drift.md](drift.md).

These are the category's defaults, not bans. A brief that explicitly chooses one
has earned it — the point is that reaching for one *when the axis was free*
means nobody was deciding. One rule (`kicker-above-heading`) is a hard ban and
says so.

Columns: **tell** = what it looks like · **why** = why it reads as AI ·
**direction** = the way out, never code · **tier** = `certain` / `judgment` /
`advisory` · **detect** = `static` (findable in source) / `render` (needs a
painted page).

## 1. Container tells

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `side-tab` | Thick coloured border on one side of a card or list row | The single most recognisable generated-UI signature; it decorates rather than communicates | Drop it, or carry the meaning in a real status element | certain | static |
| `border-accent-on-rounded` | That same thick accent border on a rounded card | The straight stripe fights the corner radius — nobody drawing this would keep both | Remove the border or the radius | certain | static |
| `nested-cards` | A card inside a card | Depth used as a substitute for hierarchy; always wrong | Flatten — spacing, type, and dividers do this better | certain | static |
| `card-grid-as-structure` | Same-size icon + heading + text cards as the page structure | The lazy container. It fits any content, which is why it says nothing about this content | Let the content shape set the layout | judgment | static |
| `icon-tile-stack` | Rounded tile with an icon in it, repeated down the page | Filler geometry; the tile carries no information the icon lacked | Drop the tile, or earn it with a real grouping job | judgment | static |
| `hero-metric-template` | Big number, small label, supporting stats, accent colour | A shape reached for before anyone asked whether the number matters | Show the number only where it drives a decision | judgment | static |

## 2. Palette

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `gradient-text` | A gradient fill on a heading or metric | Decoration standing in for emphasis | Emphasis comes from weight or size | certain | static |
| `ai-color-palette` | Purple/violet gradients, or cyan on near-black | The most recognisable generated palettes; they belong to no product | Choose a palette that comes from the subject | certain | static |
| `cream-palette` | Warm cream or beige page ground | The default "tasteful" surface, reached for by reflex rather than chosen | Pick the ground from the use scene, not the safe warm off-white | judgment | static |
| `dark-glow` | Zero-offset coloured halo used as a shadow | A glow is not depth. Real shadows have an offset and a soft blur | Give it an offset and a blur, or drop it | certain | static |
| `radial-halo` | Large radial gradient blooming behind a hero | Atmosphere applied to a page that has not earned it | Remove it; if the section needs depth, layer real surfaces | judgment | static |
| `glass-as-decoration` | Backdrop blur because it looks modern | Blur is a layering effect. Used without something to layer over, it is costume | Keep it only where content genuinely passes beneath | judgment | static |

## 3. Named default clusters

Three complete looks that generate together. Any one component is unremarkable;
the cluster arriving intact is the tell.

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `cluster-cream-serif-terracotta` | Cream ground (classically `#F4F1EA`) + serif display + terracotta accent | An entire aesthetic that arrives pre-assembled, chosen by nobody | Keep at most one leg of it, and only for a stated reason | judgment | static |
| `cluster-nearblack-acid` | Near-black ground + acid-green or vermilion accent | The technical-product costume; identical across unrelated products | Derive the accent from the subject instead | judgment | static |
| `cluster-broadsheet` | Hairline rules, zero radius, dense columns, editorial cosplay | Newspaper signifiers on a product that is not a newspaper | Borrow editorial density only where the content is genuinely editorial | judgment | static |

## 4. Type

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `flat-type-hierarchy` | Sizes too close together; no obvious step | Hierarchy asserted in markup but never made visible | Fewer sizes, bigger jumps between them | judgment | static |
| `oversized-h1` | Display type larger than the page can carry | Scale used as a substitute for having something to say | Size the headline to the content beneath it | judgment | render |
| `extreme-negative-tracking` | Tracking cranked in past legibility | Premium applied as a slider, not a decision | Back off to where the word shapes survive | certain | static |
| `system-display-face` | Impact, Arial Black, or the platform sans as the display voice | The closest installed font is a failure, not a fallback | Source and self-host a face whose character matches the intent | certain | static |
| `italic-serif-display` | Italic serif dropped in for instant editorial feel | One glyph style doing the work of an actual direction | Let the whole type system carry the voice, or drop the italic | judgment | static |

## 5. Page furniture

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `kicker-above-heading` | Tiny uppercase letter-spaced label sitting above a heading | **Hard ban — no brief earns this back.** The heading already carries its own weight | Delete the label; let the heading speak | certain | static |
| `hero-eyebrow-chip` | The same label rendered as a pill chip | Same tell wearing a different shape | Delete it, or make it a real navigation breadcrumb | certain | static |
| `numbered-section-labels` | 01 / 02 / 03 above sections | Sequence implied where none exists | Keep only when the order is information the reader needs | judgment | static |
| `modal-without-need` | A modal for a task needing neither interruption nor protected focus | The dialog reached for by default rather than chosen | Inline it, or give it a page | judgment | static |

## 6. Motion

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `bounce-easing` | Overshoot easing on ordinary UI transitions | Playfulness applied uniformly, which is the opposite of authored | Exponential ease-out unless the thing literally bounces | certain | static |
| `pulsing-dot` | A dot pulsing to signal live | Ambient animation that never resolves; costs attention forever | State it in words, or animate only on change | certain | static |
| `blinking-cursor` | A fake terminal caret blinking in decorative copy | Terminal cosplay | Remove it | certain | static |
| `marquee` | Auto-scrolling strip of logos or words | Motion with no reader in mind; unreadable by design | Make it a static row | certain | static |
| `scattered-entrances` | The same entrance animation on every section on scroll | Motion applied by rule rather than authored once | One authored moment per page, from an already-visible default | judgment | render |

## 7. Substitution — a cheap stand-in for the real thing

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `emoji-as-icon` | Unicode glyphs or emoji where an icon belongs | Icons are drawn. Emoji render differently on every platform and match no stroke or weight | Use a real icon library or authored SVG, one consistent stroke | certain | static |
| `monospace-as-costume` | Monospace for technical feel, not for code, data, or measurement | A typeface signalling a genre instead of doing a job | Reserve it for things actually aligned or literal | judgment | static |
| `sparkline-as-content` | Sparklines, progress rings, or soft-shadowed rounded rectangles standing in for content | Placeholder shapes shipped as if they were information | Show the real thing, or show nothing | judgment | static |
| `geometric-mask` | Circle, polygon, or radial cutout approximating a photographic subject edge | The cheap version of the effect; reads worse than omitting it | Derive a real alpha matte, or produce a cut-out asset | certain | static |
| `shape-assembled-illustration` | An illustration built from primitive shapes | Assembly standing in for drawing | Commission or omit; a blank space beats assembled geometry | judgment | static |

## 8. Copy

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `aphoristic-cadence` | Short declarative fragments with weighty rhythm. Like this one. | Cadence performing depth where a sentence would do | Write it as a sentence and see if it survives | judgment | static |
| `announced-restraint` | Copy that names its own design quality — "thoughtfully designed", "no clutter", "just the essentials" | A page describing its own taste is the strongest evidence it lacks it | Delete the claim; let the surface demonstrate it | judgment | static |
| `marketing-buzzword` | Seamless, elevate, unleash, supercharge, effortless | Vocabulary shared by every generated landing page | Say what it does, in the product own words | advisory | static |
| `em-dash-overuse` | Em dashes carrying most of the sentence structure | A cadence tell rather than a design one, but it travels with the rest | Vary the punctuation | advisory | static |

## 9. Rhythm

| id | tell | why | direction | tier | detect |
|---|---|---|---|---|---|
| `monotonous-spacing` | Every gap the same value | Spacing applied as a constant instead of grouping anything | Tight within a group, generous between groups; more space above a heading than below | judgment | render |
| `uniform-section-shell` | Every section the same height, padding, and shape | A page assembled from one repeated shell | Let sections differ where their content differs | judgment | render |

## Adding a rule

Only if it passes the absolute test: **would this still be slop in someone
else's app?** If it names a value, a token, a component, or a palette this
project happens to use, it is drift — put a pointer in [drift.md](drift.md)
instead.

A rule needs all seven schema fields, and `why` is the one that separates a rule
from a preference: if you cannot say why it reads as machine-made, it is taste,
and taste belongs to the owner.
