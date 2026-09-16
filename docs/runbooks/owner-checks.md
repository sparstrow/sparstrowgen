# Your open checks, in one sitting

**Why this needs you:** each check needs your own signed-in account, your installed sparstrowgen, your
judgement, or a Windows sign-out or new Windows account, none of which the agent may do. Everything
the agent could check with its testing account is already done ([`Unverified.md`](../Unverified.md)
U-5, U-12, U-14, and all but the panels of U-3 and U-4).

**What is already done:** the features are built and on production. This is only watching them work.

**How to use it:** start a session and say *"walk me through docs/runbooks/owner-checks.md"*. Do the
steps; tell the agent what you saw. The agent reads your computer's log where a step says so, marks
each check in `Unverified.md` with the date and what was seen, and logs anything wrong in
`Bugs.md`. Which checks are still open lives in `Unverified.md`, not here.

About an hour. Do them in this order: 4 and 5 install versions on your computer, and 6 and 7 end the
session.

---

## 1 — Add computer on your connected computer (U-4) · 2 min

1. Open **Machines** on app.sparstrow.com. Your computer shows **Online**.
2. Choose **Add computer**, and let the browser open sparstrowgen.
3. **Passing:** a message says **"This computer is already connected"**, Machines still lists your
   computer once, and it stays Online. The agent confirms its log has no reconnect.

## 2 — Not now on a real pairing (U-3) · 5 min

This one ends your pairing on purpose, and step 5 brings it back.

1. In **Machines**, open your computer and **disconnect** it. It leaves the list.
2. Choose **Add computer**, and let the browser open sparstrowgen.
3. **Approve this computer?** appears. Choose **Not now**.
4. **Passing:** the panel closes at once, and after a refresh Machines lists no computer.
5. Choose **Add computer** again, then **Approve computer**. It shows **Online** within a few seconds.

## 3 — Settings → Updates reads the way you want (U-6) · 5 min

1. Open **Settings → Updates**. Read your computer's card: version, Automatic updates, Check for
   updates.
2. Choose **Check now**. Expect **"You're on the latest version."**
3. Turn **Automatic updates off**, and leave it off for step 4.
4. **Passing:** nothing you would change, or tell the agent what to change. It records that as
   feedback, not a bug.

## 4 — An update waits for a running agent turn (U-9) · 15 min

1. Tell the agent to **publish the next daemon release** (a version bump, through
   `daemon-release.md`). Wait until it says the release is out.
2. In **Settings → Updates**, choose **Check now**. Expect **"v… is available."** and **Update now**.
3. In **Chat**, start a turn that takes a few minutes, for example *"Read every file in this folder and
   summarise each in a paragraph"* in a folder with some files.
4. While it is still working, go back to **Settings → Updates** and choose **Update now**.
5. **Passing:** it says **"v… is ready. It installs after the agent task running on this computer
   finishes."**, the Chat turn completes normally with its answer, and only then does the card show
   **Installing…**, then the new version and Online.
6. Turn **Automatic updates** back on.

## 5 — The message after an update is put back (rest of U-8) · 10 min

The agent has already proven the put-back itself on your computer (2026-09-14). What is left is
seeing its message on your account.

1. Tell the agent to **re-run the U-8 rollback test on your computer**. It asks your permission first,
   because it installs test builds that no other computer follows.
2. When the agent says the bad build was put back, open **Settings → Updates** before it restores
   anything.
3. **Passing:** your computer's card says **"v… did not reconnect within 2 minutes, so v… was put back
   and is running"**, and the computer is Online.
4. The agent then puts your computer back on the official release, and you see that version there.

## 6 — Your computer comes back after a Windows sign-out (U-1) · 5 min

Signing out closes the Claude app too, so do this near the end.

1. Check **Machines** shows your computer **Online**. Note the time.
2. Sign out of Windows, then sign back in. **Start nothing.**
3. After a minute, open app.sparstrow.com in your browser.
4. **Passing:** Machines shows your computer **Online**. In the next session, tell the agent the time
   you signed back in; it confirms the log has a `connected` line after it, from a copy that started
   at sign-in.

## 7 — A Windows account that never had sparstrowgen (U-2) · 15 min

1. Make a second Windows account: **Settings → Accounts → Other users → Add account**, then **I don't
   have this person's sign-in information → Add a user without a Microsoft account**.
2. Sign in to that account (**Start → your picture → the new account**).
3. In a browser, sign in to app.sparstrow.com as yourself and open **app.sparstrow.com/install**.
   Choose **Download for Windows** and open the installer. If Windows says it protected your PC,
   choose **More info**, then **Run anyway**.
4. In **Machines**, choose **Add computer**, then **Approve computer**. It appears as a second computer,
   probably with the same name as your first.
5. In **Chat**, start a new conversation and send a short message. Chat has no choice of computer: it
   uses the one that connected most recently, which is this new one, so the folder list shows that
   account's folders (`C:\Users\<new account>`). Pick any of them.
6. **Passing:** that computer shows **Online** with its agents, and the message gets an answer. The
   agent needs this only if the Chat turn fails: in that account,
   `%LOCALAPPDATA%\sparstrowgen\logs\daemon.log`.
7. Clean up: disconnect that computer in **Machines**, and Chat goes back to your usual one. Removing
   the Windows account is up to you. The test conversation stays, but its folder is on the removed
   computer, so leave it be.

## 8 — Two tabs agree on your appearance (U-15) · 2 min

Nothing to install, so this one fits anywhere.

1. Open sparstrowgen in **two tabs**: one on **Machines**, one on **Settings → Appearance**.
2. In the Appearance tab, choose a different **accent**.
3. Click back to the **Machines** tab.
4. **Passing:** it takes the new accent within a second of being looked at, with no reload.

---

## For the agent running this

- Before step 1, confirm from `%LOCALAPPDATA%\sparstrowgen\logs\daemon.log` that his computer is
  connected, and which version it runs.
- Step 4: publish through [`daemon-release.md`](daemon-release.md). Nothing else is needed; it is a
  release like any other.
- Step 5: repeat what U-8 records: a pre-release no real computer follows, never marked latest. Its
  test base version must be higher than the release step 4 published. Ask before touching his
  computer, and restore it to the official latest release afterwards.
- Mark each check as you go, with what was actually seen (`Unverified.md`'s rules). A step that fails
  becomes a bug in `Bugs.md`, linked from the check.
