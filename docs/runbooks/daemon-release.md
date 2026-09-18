# Publishing a daemon release

Installed computers update from the GitHub release marked **latest**. GitHub Actions builds and
publishes it when a version tag is pushed: there is no key to keep and nothing to run on anyone's PC
(docs/Decisions.md D-035).

There are two streams (D-037). **Stable** is that latest release, which every installed computer
reads. **Candidate** is a release the owner is testing, published as a prerelease and never marked
latest; only a computer explicitly put on the candidate channel sees it. Which stream a computer
follows is decided on that computer, not built into the executable — which is what lets a candidate
become the stable release without being rebuilt.

## Each release

1. Merge the change to `main`.
2. Pick the version: `x.y.z`, higher than the latest `daemon-v*` release. Never reuse one.
3. Tag the merged commit, with what changed as the message, and push the tag:

   ```powershell
   git tag -a daemon-v0.2.3 <commit on main> -m "One or two sentences on what changed, for the release notes."
   git push origin daemon-v0.2.3
   ```

4. The **Daemon release** workflow (`.github/workflows/daemon-release.yml`) refuses a tag that is not
   on `main` or not higher than the latest release, runs the daemon and update tests on Windows,
   builds, and publishes `sparstrowgen-setup.exe`, its `.sha256` and `sparstrowgen-update.json` as
   the latest release with notes. Follow it with `gh run watch` or in the repository's Actions tab.
5. Check what computers will see: download
   `releases/latest/download/sparstrowgen-update.json` and `sparstrowgen-setup.exe`, and confirm the
   manifest's `version` and `sha256` match the release.
6. A connected computer with automatic updates on installs it within the hour, once no agent work is
   running. **Check now** in Settings → Updates shows it at once.

Trying a build without publishing: `.\scripts\package-windows.ps1 -Version x.y.z -Output <folder>`.

## A release candidate (phase 2)

Same steps, with `daemon-candidate-v` in place of `daemon-v`:

```powershell
git tag -a daemon-candidate-v0.3.0 <commit on main> -m "What changed, for the release notes."
git push origin daemon-candidate-v0.3.0
```

The workflow publishes it as a **prerelease**, never latest, and then aims the moving
`daemon-candidate` pointer at it — the only URL candidate computers read. The version still has to be
higher than the latest stable release: a candidate that could not be promoted is not worth testing.

**Putting a computer on the candidate channel** — the staging computer, not a real one:

```powershell
sparstrowgen-setup.exe install -channel candidate
```

It is written in that computer's own data directory, so it survives updates, and it is deliberately
not offered anywhere in the product. An agent's test daemon can use `SPARSTROWGEN_CHANNEL=candidate`
instead of installing. Set it back with `install -channel stable`.

## Promoting a candidate to stable — the owner's gate

After the owner has tested the candidate on staging, he runs the **Promote a daemon candidate**
workflow (Actions tab) with the version. It **does not rebuild**: it downloads the candidate's own
`sparstrowgen-setup.exe`, checks it against both its published `.sha256` and the `sha256` inside the
manifest candidate computers verified their download against, and republishes those exact bytes as
`daemon-vx.y.z` marked latest. Only the manifest is rewritten, to name the installer at its new URL.

A rebuild from the same commit would produce a different file, which is a different candidate from
the one that was tested — `release-workflow.md` forbids it, and this is how that rule is kept rather
than remembered. Agents may prepare a candidate and its evidence; running the promotion is the
owner's. Adding a required reviewer to the `production` environment in the repository's settings
turns that from a sentence here into something GitHub enforces.

## When something goes wrong

- **The workflow fails before publishing.** Nothing reached any computer. Read the run's log, fix it
  through a pull request, then delete the tag (`git push --delete origin daemon-vX.Y.Z` and
  `git tag -d daemon-vX.Y.Z`) and tag the fixed commit.
- **A published release is broken.** A computer whose new version cannot reconnect within two minutes
  puts its old version back by itself and does not retry that version. Publish a fixed, higher
  version. Computers never install a lower version, so deleting the release does not undo it.
- **One computer is broken anyway.** Download the installer from the install page and open it on that
  computer. It keeps its pairing, so nothing else is needed.
- **Nothing to back up.** What protects updates is the GitHub account: keep two-factor sign-in on
  every account with write access to the repository.

## Moving off the signing key — once, 2026-09-14

0.2.0 and 0.2.1 accept only a manifest signed with the key made on 2026-09-13, which is on the owner's
PC at `%APPDATA%\sparstrowgen-release\update-signing.key`. So 0.2.2 is the one release that also
carries `sparstrowgen-update.json.sig`:

1. Tag and push `daemon-v0.2.2` as above. The workflow publishes it without a signature.
2. On the owner's PC, sign that release's manifest with the releasetool from before D-035, and upload
   only the signature:

   ```powershell
   git archive ba41110 server | tar -x -C <scratch>
   go -C <scratch>\server build -o <scratch>\releasetool.exe ./cmd/releasetool
   gh release download daemon-v0.2.2 --pattern sparstrowgen-setup.exe --pattern sparstrowgen-update.json --dir <scratch>\published
   <scratch>\releasetool.exe manifest -key "$env:APPDATA\sparstrowgen-release\update-signing.key" -exe <scratch>\published\sparstrowgen-setup.exe -version 0.2.2 -url https://github.com/sparstrow/sparstrowgen/releases/download/daemon-v0.2.2/sparstrowgen-setup.exe -out <scratch>\signed
   ```

   The signature covers exact bytes, so first confirm `<scratch>\signed\sparstrowgen-update.json` has
   the same SHA-256 as the published one. Then
   `gh release upload daemon-v0.2.2 <scratch>\signed\sparstrowgen-update.json.sig`.
3. Once the owner's computer reports 0.2.2, and he confirms, delete the key file. No release after
   0.2.2 needs it.

A computer still on 0.2.0 or 0.2.1 after that says it could not check for updates, and needs the
installer opened once.

**Done 2026-09-14.** 0.2.2 carried the `.sig`, and the owner's PC moved 0.2.1 → 0.2.2 at 09:19 and then
installed 0.2.3, which has no signature, at 09:22 (Unverified U-13). With the owner's confirmation the
key file and its folder were deleted the same morning. There is no key anywhere any more.

## Never

- Mark a test or pre-release build as latest: every installed computer would install it. Publish it
  as a candidate instead — that is what the channel is for.
- Rebuild a candidate to promote it. Promote the bytes that were tested, or test the new bytes.
- Publish a manifest on a release other than the one holding the installer it names.
- Raise `MinDaemonProtocol` before a release those computers can update to has been published.
