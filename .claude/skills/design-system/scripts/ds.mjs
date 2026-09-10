#!/usr/bin/env node
/**
 * ds.mjs — the design-system CLI.
 *
 * Zero dependencies, Node 18+. It has to run in any repo regardless of stack,
 * so it uses nothing but node: builtins. Do not add a package.json here.
 *
 *   node ds.mjs init  --root design-system --name "App" [--mode mirror|greenfield]
 *   node ds.mjs build --root design-system            → writes index.html
 *   node ds.mjs check --root design-system            → drift report, exit 1 on drift
 *   node ds.mjs add   --root design-system --kind component|guideline|prototype
 *                     --name Button [--group forms] [--category "ERP App"]
 *   node ds.mjs watch --root design-system            → rebuild on change
 *
 * The folder convention this walks is documented in
 * references/file-conventions.md — read that before changing anything here.
 */

import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

// ─── small helpers ────────────────────────────────────────────────────────────

const read = (p) => fs.readFileSync(p, "utf8");
const exists = (p) => fs.existsSync(p);
const write = (p, s) => {
  fs.mkdirSync(path.dirname(p), { recursive: true });
  fs.writeFileSync(p, s, "utf8");
};
const listDir = (p) =>
  exists(p) ? fs.readdirSync(p, { withFileTypes: true }) : [];
const sha = (s) => "sha256:" + createHash("sha256").update(s).digest("hex").slice(0, 16);
const titleCase = (s) =>
  s.replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());

/**
 * Every path stored in system.json or compared against it goes through here.
 * Windows hands back `a\b\c` from path.relative while the manifest is written
 * with forward slashes, and comparing the two raw is how a correctly-registered
 * card gets reported as unmirrored drift.
 */
const posix = (p) => p.split(path.sep).join("/");
/** Relative POSIX path from one file's directory to another file. */
const relFrom = (fromFile, toFile) => posix(path.relative(path.dirname(fromFile), toFile));

/**
 * A card's own relative hrefs (e.g. `../../styles.css`) are written correctly
 * for the card's REAL file location. But `build` injects every card into
 * index.html via `iframe.srcdoc`, and a srcdoc document resolves relative URLs
 * against the PARENT page's location (index.html, at the system root) — not
 * against the original file the HTML came from. Without correction, a card
 * two levels deep resolves `../../styles.css` from index.html's own location
 * and walks straight out of the design-system folder.
 *
 * The fix is a `<base>` tag, which a browser resolves against the *fallback*
 * base (the parent document's URL) and then uses for every other relative
 * URL in the document from that point on — re-anchoring the card back to
 * where it actually lives, whether index.html was opened via file:// or
 * served over http. Verified both ways with the URL constructor before
 * relying on it here; do not remove without re-checking both.
 */
function rebaseCard(html, cardDir, root) {
  const base = posix(path.relative(root, cardDir)) + "/";
  if (/<head[\s>]/i.test(html)) {
    return html.replace(/<head([\s>])/i, `<head$1<base href="${base}">`);
  }
  // No <head> found (a hand-edited or unusual card) — prepending still works;
  // browsers hoist a leading <base> into an implicit <head>.
  return `<base href="${base}">\n${html}`;
}

/*
 * KNOWN LIMITATION — one 404 per card in the console, then the correct 200.
 *
 * The preload scanner starts fetching a card's subresources before the parser
 * has applied the <base> above, so `../styles.css` is first resolved against
 * index.html's own location and misses, then re-resolved correctly. Cards
 * render fine; the cost is console noise on a page whose job is to be
 * inspected, plus one wasted request each.
 *
 * Rewriting the hrefs to their post-base form was tried (2026-08-19) and is
 * WRONG: <base> then re-anchors the already-resolved URL a second time and
 * every card loses its stylesheet. If you attempt this again, the rewrite and
 * the <base> tag have to be removed together, and that means also rewriting
 * relative URLs inside inline CSS `url(...)`, which <base> currently covers
 * for free.
 *
 * Do not "fix" it with server-absolute URLs: `index.html` is required to work
 * when opened directly over file://, which is the whole reason it is the
 * primary view.
 */

/** Recursively collect files matching a suffix, returning repo-relative paths. */
function walk(dir, suffix, out = []) {
  for (const e of listDir(dir)) {
    const full = path.join(dir, e.name);
    if (e.isDirectory()) walk(full, suffix, out);
    else if (e.name.endsWith(suffix)) out.push(full);
  }
  return out;
}

// ─── markdown → html ──────────────────────────────────────────────────────────
// Deliberately small. Usage notes lean on code fences and tables (a variant →
// semantic-status mapping table is the single most useful thing a .prompt.md
// carries), so those two get real support; everything else is best-effort.

function escapeHtml(s) {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function inline(s) {
  return escapeHtml(s)
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(^|[^*])\*([^*]+)\*/g, "$1<em>$2</em>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
}

function markdown(src) {
  const lines = src.split(/\r?\n/);
  const out = [];
  let i = 0;
  let listType = null;

  const closeList = () => {
    if (listType) {
      out.push(`</${listType}>`);
      listType = null;
    }
  };

  while (i < lines.length) {
    const line = lines[i];

    // fenced code
    if (/^```/.test(line)) {
      closeList();
      const lang = line.slice(3).trim();
      const buf = [];
      i++;
      while (i < lines.length && !/^```/.test(lines[i])) buf.push(lines[i++]);
      i++;
      out.push(
        `<pre class="code"${lang ? ` data-lang="${escapeHtml(lang)}"` : ""}><code>${escapeHtml(
          buf.join("\n"),
        )}</code></pre>`,
      );
      continue;
    }

    // table — needs a header row followed by a |---| separator
    if (/^\s*\|/.test(line) && /^\s*\|[\s:|-]+\|\s*$/.test(lines[i + 1] || "")) {
      closeList();
      const cells = (r) =>
        r.trim().replace(/^\||\|$/g, "").split("|").map((c) => c.trim());
      const head = cells(line);
      i += 2;
      const body = [];
      while (i < lines.length && /^\s*\|/.test(lines[i])) body.push(cells(lines[i++]));
      out.push(
        `<table><thead><tr>${head.map((h) => `<th>${inline(h)}</th>`).join("")}</tr></thead>` +
          `<tbody>${body
            .map((r) => `<tr>${r.map((c) => `<td>${inline(c)}</td>`).join("")}</tr>`)
            .join("")}</tbody></table>`,
      );
      continue;
    }

    const heading = /^(#{1,4})\s+(.*)$/.exec(line);
    if (heading) {
      closeList();
      const level = heading[1].length;
      out.push(`<h${level}>${inline(heading[2])}</h${level}>`);
      i++;
      continue;
    }

    const bullet = /^\s*([-*+])\s+/.test(line);
    const numbered = /^\s*\d+\.\s+/.test(line);
    if (bullet || numbered) {
      const want = bullet ? "ul" : "ol";
      if (listType !== want) {
        closeList();
        out.push(`<${want}>`);
        listType = want;
      }
      // A wrapped list item continues on following lines. Absorb them, or the
      // remainder of the sentence escapes the <li> and renders as a stray
      // paragraph below the list.
      const buf = [line.replace(bullet ? /^\s*[-*+]\s+/ : /^\s*\d+\.\s+/, "")];
      i++;
      while (
        i < lines.length &&
        lines[i].trim() !== "" &&
        !/^(#{1,4}\s|```|\s*[-*+]\s|\s*\d+\.\s|\s*>|\s*\|)/.test(lines[i])
      ) {
        buf.push(lines[i].trim());
        i++;
      }
      out.push(`<li>${inline(buf.join(" "))}</li>`);
      continue;
    }

    if (/^\s*>\s?/.test(line)) {
      closeList();
      // Absorb every consecutive quoted line into ONE blockquote and run
      // inline() over the joined text. Processing each line separately would
      // split emphasis that spans a line break — `**bold` on one line and
      // `bold**` on the next never pair, and both markers render literally.
      const buf = [];
      while (i < lines.length && /^\s*>\s?/.test(lines[i])) {
        buf.push(lines[i].replace(/^\s*>\s?/, ""));
        i++;
      }
      out.push(`<blockquote>${inline(buf.join(" "))}</blockquote>`);
      continue;
    }

    if (/^\s*(-{3,}|\*{3,})\s*$/.test(line)) {
      closeList();
      out.push("<hr>");
      i++;
      continue;
    }

    if (line.trim() === "") {
      closeList();
      i++;
      continue;
    }

    // paragraph — greedily absorb following non-structural lines
    closeList();
    const buf = [line];
    i++;
    while (
      i < lines.length &&
      lines[i].trim() !== "" &&
      !/^(#{1,4}\s|```|\s*[-*+]\s|\s*\d+\.\s|\s*>|\s*\|)/.test(lines[i])
    ) {
      buf.push(lines[i++]);
    }
    out.push(`<p>${inline(buf.join(" "))}</p>`);
  }

  closeList();
  return out.join("\n");
}

// ─── token parsing ────────────────────────────────────────────────────────────
// Pulls `--name: value;` declarations out of a CSS file. Used both to seed
// tokens/ in mirror mode and to detect drift later.

function parseCssTokens(css) {
  const tokens = {};
  for (const m of css.matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) {
    tokens[m[1]] = m[2].trim().replace(/\s+/g, " ");
  }
  return tokens;
}

// ─── manifest ─────────────────────────────────────────────────────────────────
// system.json is what makes `check` possible. Without a recorded fingerprint of
// each mirrored source there is nothing to diff against, and "maintaining" the
// system degrades into regenerating it and hoping.

const MANIFEST = "system.json";

function loadManifest(root) {
  const p = path.join(root, MANIFEST);
  if (!exists(p)) {
    throw new Error(
      `No ${MANIFEST} in ${root}. Run \`ds.mjs init --root ${root} --name "<App>"\` first.`,
    );
  }
  return JSON.parse(read(p));
}

function saveManifest(root, m) {
  write(path.join(root, MANIFEST), JSON.stringify(m, null, 2) + "\n");
}

// ─── card collection ──────────────────────────────────────────────────────────

/**
 * A card is any `*.card.html`. Its group is the directory that holds it, its
 * usage notes are the sibling `*.prompt.md`, and a design card additionally
 * points at the `*.dc.html` prototype it previews.
 */
function collectCards(root) {
  const sections = [];

  const addSection = (dirName, label) => {
    const dir = path.join(root, dirName);
    if (!exists(dir)) return;

    // Cards sitting directly in the section dir group under the section itself;
    // cards in subdirectories group under the subdirectory name.
    const groups = new Map();
    const push = (groupLabel, card) => {
      if (!groups.has(groupLabel)) groups.set(groupLabel, []);
      groups.get(groupLabel).push(card);
    };

    for (const file of walk(dir, ".card.html")) {
      const rel = path.relative(dir, file);
      const parts = rel.split(path.sep);
      const groupLabel = parts.length > 1 ? titleCase(parts[0]) : label;
      const base = path.basename(file, ".card.html");
      const html = read(file);

      // Title/description come from the card's own <title>/meta so the card
      // stays the single source of truth for how it presents itself.
      const titleMatch = /<title>([\s\S]*?)<\/title>/i.exec(html);
      const descMatch = /<meta\s+name=["']description["']\s+content=["']([^"']*)["']/i.exec(html);

      const promptCandidates = [
        path.join(path.dirname(file), base + ".prompt.md"),
        path.join(path.dirname(file), titleCase(base).replace(/ /g, "") + ".prompt.md"),
      ];
      const promptPath = promptCandidates.find(exists);

      const dcCandidates = walk(path.dirname(file), ".dc.html").filter(
        (p) => path.basename(p, ".dc.html").toLowerCase().replace(/[^a-z0-9]/g, "") ===
          base.toLowerCase().replace(/[^a-z0-9]/g, ""),
      );

      push(groupLabel, {
        id: rel.replace(/[\\/.]/g, "-"),
        name: titleMatch ? titleMatch[1].trim() : titleCase(base),
        description: descMatch ? descMatch[1] : "",
        section: label,
        group: groupLabel,
        html: rebaseCard(html, path.dirname(file), root),
        prompt: promptPath ? markdown(read(promptPath)) : null,
        promptPath: promptPath ? path.relative(root, promptPath) : null,
        cardPath: path.relative(root, file),
        live: dcCandidates.length ? path.relative(root, dcCandidates[0]) : null,
      });
    }

    for (const [groupLabel, cards] of groups) {
      cards.sort((a, b) => a.name.localeCompare(b.name));
      sections.push({ section: label, group: groupLabel, cards });
    }
  };

  addSection("guidelines", "Foundations");
  addSection("components", "Components");
  addSection("designs", "Designs");
  return sections;
}

// ─── build ────────────────────────────────────────────────────────────────────

function cmdBuild({ root }) {
  const manifest = loadManifest(root);
  const sections = collectCards(root);
  const readmePath = path.join(root, "README.md");
  const readme = exists(readmePath) ? markdown(read(readmePath)) : "";

  const shell = read(path.join(import.meta.dirname, "..", "assets", "viewer-shell.html"));
  const payload = {
    name: manifest.name,
    mode: manifest.mode,
    theme: manifest.theme || { default: "dark" },
    generatedAt: manifest.lastBuild || null,
    readme,
    sections,
  };

  // Cards go through a JSON island rather than inline srcdoc attributes:
  // attribute-escaping a full HTML document is fragile, and each card carries
  // its own <style>, so it must render inside an iframe to stay isolated.
  // replaceAll, not replace — the name appears in both <title> and the brand.
  const html = shell
    .replaceAll("__SYSTEM_NAME__", escapeHtml(manifest.name))
    .replace(
      "__PAYLOAD__",
      JSON.stringify(payload).replace(/</g, "\\u003c").replace(/-->/g, "--\\u003e"),
    );

  const outPath = path.join(root, "index.html");
  write(outPath, html);

  const cardCount = sections.reduce((n, s) => n + s.cards.length, 0);
  manifest.lastBuild = new Date().toISOString();
  manifest.cardCount = cardCount;
  saveManifest(root, manifest);

  console.log(`built ${path.relative(process.cwd(), outPath)}`);
  console.log(`  ${cardCount} cards across ${sections.length} groups (${manifest.mode} mode)`);
  return 0;
}

// ─── check (drift detection) ──────────────────────────────────────────────────

function cmdCheck({ root }) {
  const manifest = loadManifest(root);
  const findings = [];

  // 1. token drift — the design system claims a value the source no longer has
  for (const src of manifest.sources?.tokens || []) {
    if (!exists(src)) {
      findings.push({ kind: "missing-source", detail: `token source is gone: ${src}` });
      continue;
    }
    const current = parseCssTokens(read(src));
    const recorded = manifest.tokens || {};
    for (const [name, value] of Object.entries(recorded)) {
      if (!(name in current)) {
        findings.push({ kind: "token-removed", detail: `${name} no longer defined in ${src}` });
      } else if (current[name] !== value) {
        findings.push({
          kind: "token-changed",
          detail: `${name}: system says "${value}", ${src} says "${current[name]}"`,
        });
      }
    }
    for (const name of Object.keys(current)) {
      if (!(name in recorded)) {
        findings.push({ kind: "token-added", detail: `${name} exists in ${src} but not in the system` });
      }
    }
  }

  // 2. mirrored component drift — the real source changed since we documented it
  for (const m of manifest.mirrors || []) {
    if (!exists(m.source)) {
      findings.push({
        kind: "dangling-mirror",
        detail: `${m.card} documents ${m.source}, which no longer exists`,
      });
      continue;
    }
    const now = sha(read(m.source));
    if (now !== m.fingerprint) {
      findings.push({
        kind: "component-changed",
        detail: `${m.source} changed since ${m.card} was written (re-read it and update the usage notes)`,
      });
    }
  }

  // 3. orphan cards — present on disk, absent from the manifest's mirror list.
  //    Only meaningful in mirror mode; greenfield cards have no upstream source.
  if (manifest.mode === "mirror") {
    const known = new Set((manifest.mirrors || []).map((m) => posix(m.card)));
    for (const file of walk(path.join(root, "components"), ".card.html")) {
      const rel = posix(path.relative(root, file));
      if (!known.has(rel)) {
        findings.push({
          kind: "unmirrored-card",
          detail: `${rel} has no source recorded in ${MANIFEST} — drift here is invisible`,
        });
      }
    }
  }

  // 4. prototypes whose preview card went missing (or vice versa)
  for (const dc of walk(path.join(root, "designs"), ".dc.html")) {
    const sibling = dc.replace(/\.dc\.html$/, ".card.html");
    const looseMatch = walk(path.dirname(dc), ".card.html").length > 0;
    if (!exists(sibling) && !looseMatch) {
      findings.push({
        kind: "prototype-uncarded",
        detail: `${path.relative(root, dc)} has no preview card, so it will not appear in the index`,
      });
    }
  }

  if (!findings.length) {
    console.log(`✓ no drift — ${root} matches its sources`);
    return 0;
  }

  console.log(`${findings.length} drift finding(s) in ${root}:\n`);
  const byKind = {};
  for (const f of findings) (byKind[f.kind] ||= []).push(f.detail);
  for (const [kind, details] of Object.entries(byKind)) {
    console.log(`  ${kind}`);
    for (const d of details) console.log(`    - ${d}`);
    console.log("");
  }
  console.log("Fix by updating the affected cards/usage notes, then re-run `ds.mjs sync`.");
  return 1;
}

// ─── sync (re-record fingerprints after a deliberate update) ──────────────────

function cmdSync({ root }) {
  const manifest = loadManifest(root);
  let updated = 0;

  for (const src of manifest.sources?.tokens || []) {
    if (!exists(src)) continue;
    manifest.tokens = parseCssTokens(read(src));
    updated++;
  }
  for (const m of manifest.mirrors || []) {
    if (!exists(m.source)) continue;
    const now = sha(read(m.source));
    if (now !== m.fingerprint) {
      m.fingerprint = now;
      updated++;
    }
  }

  manifest.lastSync = new Date().toISOString();
  saveManifest(root, manifest);
  console.log(`synced ${updated} source fingerprint(s) — drift baseline reset`);
  console.log("Only do this after the cards/usage notes were actually updated to match.");
  return 0;
}

// ─── add ──────────────────────────────────────────────────────────────────────

function cmdAdd({ root, kind, name, group, category, source }) {
  if (!name) throw new Error("--name is required");
  const manifest = loadManifest(root);
  const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  const created = [];

  // The href has to be computed from where the card actually lands, not from
  // the root — a card two directories deep needs ../../styles.css.
  const cardStub = (cardPath, title, desc, body) =>
    `<!doctype html>
<html><head><meta charset="utf-8">
<title>${escapeHtml(title)}</title>
<meta name="description" content="${escapeHtml(desc)}">
<link rel="stylesheet" href="${relFrom(cardPath, path.join(root, "styles.css"))}">
<style>
  body { margin:0; padding:var(--space-5,20px); background:var(--card,#161618); color:var(--foreground,#f4f4f5);
         font-family:var(--font-sans,system-ui); font-size:13px; }
  .row { display:grid; grid-template-columns:110px 1fr; gap:var(--space-4,16px); align-items:center;
         padding:var(--space-4,16px) 0; border-bottom:1px solid var(--border,#26262b); }
  .row:last-child { border-bottom:0; }
  .lbl { font-size:11px; font-weight:600; letter-spacing:.05em; text-transform:uppercase; color:var(--muted-fg,#8a8a93); }
  .specimens { display:flex; flex-wrap:wrap; gap:var(--space-3,12px); align-items:center; }
</style></head>
<body>
${body}
</body></html>
`;

  if (kind === "component") {
    const dir = path.join(root, "components", group || slug);
    const cardPath = path.join(dir, `${slug}.card.html`);
    write(
      cardPath,
      cardStub(
        cardPath,
        name,
        `${name} — variants and states`,
        `<div class="row"><div class="lbl">Variants</div><div class="specimens">
  <!-- Render every variant here. A card that shows only the default state
       teaches nothing the component's own source does not already say. -->
</div></div>
<div class="row"><div class="lbl">States</div><div class="specimens">
  <!-- default / hover / focus-visible / disabled / error / loading, as applicable -->
</div></div>`,
      ),
    );
    created.push(cardPath);

    const promptPath = path.join(dir, `${slug}.prompt.md`);
    write(
      promptPath,
      `# ${name}

<!-- These are the usage notes an agent reads before using this component.
     Write what the source code cannot say: which variant means what, and
     which combinations are wrong. -->

${source ? `Source of truth: \`${source}\`\n\n` : ""}## Usage

\`\`\`tsx
<${name} />
\`\`\`

## Variant → meaning

| Variant | Use for |
|---|---|
|  |  |

## Notes

-
`,
    );
    created.push(promptPath);

    if (source) {
      if (!exists(source)) throw new Error(`--source ${source} does not exist`);
      manifest.mirrors ||= [];
      manifest.mirrors.push({
        card: posix(path.relative(root, cardPath)),
        source: posix(source),
        fingerprint: sha(read(source)),
      });
    }
  } else if (kind === "guideline") {
    const cardPath = path.join(root, "guidelines", `${slug}.card.html`);
    write(
      cardPath,
      cardStub(
        cardPath,
        name,
        `${name} — foundation`,
        `<div class="row"><div class="lbl">${escapeHtml(name)}</div><div class="specimens"></div></div>`,
      ),
    );
    created.push(cardPath);
  } else if (kind === "prototype") {
    const dir = path.join(root, "designs", (category || "Designs").replace(/[\\/]/g, "-"));
    const dcPath = path.join(dir, `${slug}.dc.html`);
    write(
      dcPath,
      `<!doctype html>
<html><head><meta charset="utf-8"><title>${escapeHtml(name)}</title>
<link rel="stylesheet" href="${relFrom(dcPath, path.join(root, "styles.css"))}">
</head><body>
<!-- Full clickable prototype. See the interactive-prototype skill. -->
</body></html>
`,
    );
    created.push(dcPath);
    const cardPath = path.join(dir, `${slug}.card.html`);
    write(
      cardPath,
      cardStub(cardPath, name, `${name} — prototype preview`, `<div class="specimens"></div>`),
    );
    created.push(cardPath);
  } else {
    throw new Error(`--kind must be component | guideline | prototype (got ${kind})`);
  }

  saveManifest(root, manifest);
  for (const f of created) console.log(`created ${path.relative(process.cwd(), f)}`);
  console.log(`\nFill these in, then run \`ds.mjs build --root ${root}\`.`);
  return 0;
}

// ─── init ─────────────────────────────────────────────────────────────────────

function cmdInit({ root, name, mode, tokenSource, componentSource }) {
  if (!name) throw new Error("--name is required (the app or product name)");
  const resolvedMode = mode || (tokenSource || componentSource ? "mirror" : "greenfield");

  const manifest = {
    name,
    mode: resolvedMode,
    version: 1,
    theme: { default: "dark", toggleAttr: "data-theme" },
    sources: {
      tokens: tokenSource ? [tokenSource] : [],
      components: componentSource ? [componentSource] : [],
    },
    tokens: {},
    mirrors: [],
    created: new Date().toISOString(),
  };

  if (tokenSource && exists(tokenSource)) {
    manifest.tokens = parseCssTokens(read(tokenSource));
  }

  for (const d of ["tokens", "guidelines", "components", "designs", "lib", "assets"]) {
    fs.mkdirSync(path.join(root, d), { recursive: true });
  }

  if (!exists(path.join(root, "styles.css"))) {
    write(
      path.join(root, "styles.css"),
      `/* Entry point. @import only — real values live in tokens/. */\n` +
        `@import "./tokens/colors.css";\n` +
        `@import "./tokens/typography.css";\n` +
        `@import "./tokens/spacing.css";\n`,
    );
  }
  for (const t of ["colors", "typography", "spacing"]) {
    const p = path.join(root, "tokens", `${t}.css`);
    if (!exists(p)) write(p, `:root {\n  /* ${t} tokens */\n}\n`);
  }
  if (!exists(path.join(root, "CHANGELOG.md"))) {
    write(
      path.join(root, "CHANGELOG.md"),
      `# ${name} Design System — changelog\n\nNewest first. Record token changes, new components, and new prototypes.\n\n## ${new Date().toISOString().slice(0, 10)}\n\n- System initialised in \`${resolvedMode}\` mode.\n`,
    );
  }

  saveManifest(root, manifest);
  console.log(`initialised ${root} (${resolvedMode} mode)`);
  if (resolvedMode === "mirror") {
    console.log(`  tokens mirrored from: ${tokenSource || "(none set)"}`);
    console.log(`  ${Object.keys(manifest.tokens).length} tokens recorded`);
  }
  return 0;
}

// ─── serve ────────────────────────────────────────────────────────────────────
// index.html can be opened straight off disk, but a `.dc.html` prototype that
// loads seed data from lib/ hits the file:// origin restriction. Serving over
// HTTP makes the whole system behave the way it will in a browser for real.

async function cmdServe({ root, port }) {
  const { createServer } = await import("node:http");
  const listenPort = Number(port) || 4321;

  const TYPES = {
    ".html": "text/html; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".js": "text/javascript; charset=utf-8",
    ".mjs": "text/javascript; charset=utf-8",
    ".json": "application/json; charset=utf-8",
    ".svg": "image/svg+xml",
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".woff2": "font/woff2",
    ".md": "text/plain; charset=utf-8",
  };

  const rootAbs = path.resolve(root);

  createServer((req, res) => {
    let rel = decodeURIComponent(req.url.split("?")[0]);
    if (rel.endsWith("/")) rel += "index.html";
    const target = path.resolve(rootAbs, "." + rel);

    // Never serve outside the design-system folder, even if the URL walks up.
    if (!target.startsWith(rootAbs)) {
      res.writeHead(403).end("forbidden");
      return;
    }
    if (!exists(target) || fs.statSync(target).isDirectory()) {
      res.writeHead(404).end("not found");
      return;
    }
    res.writeHead(200, {
      "content-type": TYPES[path.extname(target)] || "application/octet-stream",
      "cache-control": "no-store",
    });
    fs.createReadStream(target).pipe(res);
  }).listen(listenPort, () => {
    console.log(`serving ${root} at http://localhost:${listenPort}/`);
  });
}

// ─── watch ────────────────────────────────────────────────────────────────────

function cmdWatch({ root }) {
  cmdBuild({ root });
  console.log(`watching ${root} … (ctrl-c to stop)`);
  let timer = null;
  fs.watch(root, { recursive: true }, (_e, file) => {
    if (!file || /index\.html|system\.json/.test(file)) return;
    clearTimeout(timer);
    timer = setTimeout(() => {
      try {
        cmdBuild({ root });
      } catch (err) {
        console.error(`build failed: ${err.message}`);
      }
    }, 120);
  });
}

// ─── entry ────────────────────────────────────────────────────────────────────

const { values, positionals } = parseArgs({
  allowPositionals: true,
  options: {
    root: { type: "string", default: "design-system" },
    name: { type: "string" },
    mode: { type: "string" },
    kind: { type: "string" },
    group: { type: "string" },
    category: { type: "string" },
    source: { type: "string" },
    "token-source": { type: "string" },
    "component-source": { type: "string" },
    port: { type: "string" },
  },
});

const cmd = positionals[0];
const args = {
  ...values,
  tokenSource: values["token-source"],
  componentSource: values["component-source"],
};

const commands = {
  init: cmdInit,
  build: cmdBuild,
  check: cmdCheck,
  sync: cmdSync,
  add: cmdAdd,
  watch: cmdWatch,
  serve: cmdServe,
};

if (!cmd || !commands[cmd]) {
  console.error(
    `usage: ds.mjs <init|build|check|sync|add|watch|serve> --root <dir> [options]\n` +
      `see .claude/skills/design-system/SKILL.md`,
  );
  process.exit(2);
}

// `watch` and `serve` stay resident — exiting on their return value would kill
// them the moment they finished setting up.
const LONG_RUNNING = new Set(["watch", "serve"]);

try {
  const result = await commands[cmd](args);
  if (!LONG_RUNNING.has(cmd)) process.exit(result ?? 0);
} catch (err) {
  console.error(`error: ${err.message}`);
  process.exit(1);
}
