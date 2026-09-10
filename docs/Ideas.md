# Ideas

Unscoped. No commitment, no decision, possibly never built. Distinct from
[`Deferred.md`](Deferred.md): those were agreed and parked, these were merely noticed.

A line or two is fine — an idea earns length only from evidence, never from speculation. Ids are
never reused.

If an idea graduates it becomes a design the owner picks from — see `AGENTS.md` §1 — and the entry
is deleted.

---

## I-1 — Remote desktop access to the machine running the daemon

**What was noticed** — 2026-09-09, while scoping the preview tunnel. The owner asked whether he
could "see the system window like a screenshare and access the desktop like a virtual computer",
and whether that should be built here or taken from an existing service.

**What is true today** — nothing is built. The daemon holds a persistent outbound connection to
the server, which is enough to reach the machine, but carries no screen or input capability.

**The reframe** — three separate capabilities were being conflated, and only one of them is this
project's work:

1. *Human remote desktop* — see and control the machine from a phone. Commodity.
2. *App preview* — see the app the agent just built, running on the machine's localhost. This is
   the reverse HTTP tunnel, and for this purpose it beats remote desktop outright: crisp text
   instead of video compression, real touch interaction, no encoding latency. **This is a planned
   feature, not an idea, and does not belong in this entry.**
3. *Agent computer-use* — the agent screenshots and clicks. Screenshot-on-demand plus input
   injection at roughly 1fps. No streaming, no encoding, no transport tuning. A far smaller
   problem, and if the target is a browser it is Playwright/CDP rather than desktop capture.

**Resolution — do not build (1).** A credible remote desktop stack means platform capture APIs,
hardware video encoders, damage-region tracking, adaptive bitrate, WebRTC with tuned jitter
buffers, scancode-correct input injection across keyboard layouts, multi-monitor and DPI scaling,
and NAT traversal with TURN relay. That is engineer-years of commodity work, and the result would
be worse than free alternatives.

**A shape, if it is ever wanted** — deploy it and embed it. MeshCentral is the closest fit: one
container in Coolify, agent dials outbound (the same trust model the daemon already uses), and a
fully browser-based viewer so a phone needs no app. Apache Guacamole over the machine's built-in
Windows RDP host is the higher-fidelity alternative, at the cost of needing a network path to the
RDP port. Either way, the daemon reports "remote desktop available", the web UI iframes the
third-party viewer, and our session layer gates access.

**What it touches** — overlaps the preview tunnel only superficially; they solve different
problems and the tunnel is not a step toward this.

**What would revive it** — a need to reach a *native* application on the machine, not a web app.
Everything currently on the roadmap is reachable through the preview tunnel instead.

*Surfaced while separating the preview tunnel from screen sharing.*
