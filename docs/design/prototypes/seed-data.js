/* Shared seed data for the prototypes. Realistic shapes, not lorem ipsum: the names, folders, providers and
   states below are the ones the real app shows. Update this file, never a copy of it inside a prototype. */
window.SPG_SEED = {
  account: { email: "agent@sparstrow.com" },
  // Separate areas of work inside one account (D-050). The owner's own split:
  // "I would create one personal and one work related workspace."
  workspaces: ["Personal", "Client work"],
  conversations: [
    { id: "c1", provider: "claude", model: "Opus 4.6", title: "Fix flaky reconnect test", folder: "sparstrowgen", updated: "2 hours ago", tokens: 24100, spendUsd: 0.42 },
    { id: "c2", provider: "codex", model: "GPT-5.6 Sol", title: "Move the invoice export to a job", folder: "billing-api", updated: "Yesterday", tokens: 8300, spendUsd: 0 },
    { id: "c3", provider: "agy", model: "Gemini 3.1 Pro", title: "", folder: "notes", updated: "Just now", tokens: 0, spendUsd: 0 },
    { id: "c4", provider: "claude", model: "Opus 4.6", title: "Why is the daemon refusing pairing?", folder: "sparstrowgen", updated: "3 days ago", tokens: 3900, spendUsd: 0.07 },
    { id: "c5", provider: "claude", model: "Sonnet 4.6", title: "Draft the release notes for daemon-v0.2.3", folder: "sparstrowgen", updated: "3 days ago", tokens: 2100, spendUsd: 0.03 },
    { id: "c6", provider: "codex", model: "GPT-5.6 Sol", title: "Rename the orders table columns to match the ERP export", folder: "billing-api", updated: "Last week", tokens: 12700, spendUsd: 0 },
    { id: "c7", provider: "agy", model: "Gemini 3.1 Pro", title: "Compare the EDI 850 mapping against the vendor spec", folder: "edi-mappings", updated: "Last week", tokens: 6800, spendUsd: 0 },
    { id: "c8", provider: "claude", model: "Opus 4.6", title: "Check why the nightly backup email did not arrive", folder: "sparstrowgen", updated: "2 weeks ago", tokens: 5200, spendUsd: 0.09 },
  ],
  archived: [
    { id: "a1", provider: "codex", model: "GPT-5.6 Sol", title: "Try the old Windows installer flow", folder: "sparstrowgen", updated: "A month ago" },
    { id: "a2", provider: "claude", model: "Opus 4.6", title: "Spike: stream partial answers", folder: "sparstrowgen", updated: "A month ago" },
  ],
  transcript: {
    c2: [
      { role: "user", text: "Move the invoice export out of the request path and into a nightly job.", at: "Yesterday" },
      { role: "agent", provider: "codex", model: "GPT-5.6 Sol", at: "Yesterday", text: "Done. The export now runs from a scheduled job at 02:00 and writes the file to the shared folder; the request handler only enqueues it. The existing export button still works and reports that the file will be ready shortly." },
    ],
    c4: [
      { role: "user", text: "Why is the daemon refusing pairing?", at: "3 days ago" },
      { role: "agent", provider: "claude", model: "Opus 4.6", at: "3 days ago", text: "The pairing request was already claimed by another computer, so the server refuses a second claim until the first is approved or declined. Decline it from Machines and start again." },
    ],
    c5: [
      { role: "user", text: "Draft the release notes for daemon-v0.2.3.", at: "3 days ago" },
      { role: "agent", provider: "claude", model: "Sonnet 4.6", at: "3 days ago", text: "Fixed: a computer no longer goes offline after signing out of Windows and in again. Improved: the update check now waits for running agent work to finish before installing." },
    ],
    c6: [
      { role: "user", text: "Rename the orders table columns to match the ERP export.", at: "Last week" },
      { role: "agent", provider: "codex", model: "GPT-5.6 Sol", at: "Last week", text: "Renamed nine columns in one migration and updated the queries that read them. The export now matches the ERP field names one for one; the old names are gone, so anything outside this repository that reads them needs the same change." },
    ],
    c7: [
      { role: "user", text: "Compare the EDI 850 mapping against the vendor spec.", at: "Last week" },
      { role: "agent", provider: "agy", model: "Gemini 3.1 Pro", at: "Last week", text: "Two segments differ: the ship-to qualifier is mapped to N1*ST but the vendor expects N1*SH, and the requested delivery date is read from DTM*002 instead of DTM*106. Everything else matches." },
    ],
    c8: [
      { role: "user", text: "Check why the nightly backup email did not arrive.", at: "2 weeks ago" },
      { role: "agent", provider: "claude", model: "Opus 4.6", at: "2 weeks ago", text: "The backup finished, but the notification step is disabled: the instance email settings are empty, so nothing was sent. Fill them in and the next run will report." },
    ],
    c1: [
      { role: "user", text: "The reconnect test fails on CI but passes on my machine. Why?", at: "10:41" },
      { role: "agent", provider: "claude", model: "Opus 4.6", at: "10:42", text: "It waits a fixed 200 ms for the socket to reopen. CI is slower, so the assertion runs before the reconnect lands. Wait for the `connected` event instead of sleeping, and the test stops depending on how fast the machine is." },
      { role: "user", text: "Show me the change.", at: "10:44" },
      { role: "agent", provider: "claude", model: "Opus 4.6", at: "10:44", text: "In `daemon_test.go`, replace `time.Sleep(200 * time.Millisecond)` with `<-client.Connected()` guarded by a two second timeout. Nothing else in the test needs to change." },
    ],
  },
  providers: [
    { id: "claude", label: "claude", availability: "available", headroom: "5-hour resets 4h 12m" },
    { id: "codex", label: "codex", availability: "available", headroom: "no limit data" },
    { id: "agy", label: "agy", availability: "blocked", reason: "Not installed" },
  ],
  machines: [
    { id: "m1", name: "SRIHARI-DESKTOP", online: true, version: "0.2.3", providers: [
      { id: "claude", label: "claude", availability: "available" },
      { id: "codex", label: "codex", availability: "available" },
      { id: "agy", label: "agy", availability: "blocked", reason: "Not installed" } ] },
    { id: "m2", name: "Agent test computer", online: false, version: "0.2.3", providers: [
      { id: "claude", label: "claude", availability: "waitable", reason: "Machine unreachable" } ] },
  ],
  settings: [
    { group: "Account", pages: [ { id: "account", label: "Account", icon: "circle-user" }, { id: "password", label: "Password", icon: "key-round" }, { id: "appearance", label: "Appearance", icon: "palette" } ] },
    { group: "Workspaces", pages: [ { id: "workspaces", label: "Workspaces", icon: "folder-open" } ] },
    { group: "Computers", pages: [ { id: "updates", label: "Updates", icon: "download" } ] },
  ],
};
