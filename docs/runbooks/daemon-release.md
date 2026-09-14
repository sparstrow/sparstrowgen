# Publishing a daemon release

Installed computers update from the GitHub release marked **latest**, and only if its manifest is
signed with the update signing key (docs/Decisions.md D-034). This is how a release is made.

## Once: the update signing key

The key was generated on the release machine (the owner's PC) on 2026-09-13 at
`%APPDATA%\sparstrowgen-release\update-signing.key`. Every release from 0.2.0 on trusts its public half.

**Owner action — back it up.** Copy that file to somewhere private that is not this repository and
not a shared drive: a password manager's secure file, or an encrypted USB drive. Never commit it,
paste it into chat, or upload it anywhere public.

- **If it is lost:** installed computers can no longer verify a new release, so they stay on their
  version. Recovery is a new key (`releasetool keygen` refuses to overwrite, so move the old file
  away first) and a release that every computer installs by hand once, from the install page.
- **If it leaks:** someone who can also get a computer to download from them could install code on
  it. Recover the same way, and treat it as urgent.

Creating it again on a new release machine, only when there is no key at all:

```powershell
go -C server run ./cmd/releasetool keygen -key "$env:APPDATA\sparstrowgen-release\update-signing.key"
```

## Each release

1. Pick the version: `x.y.z`, higher than the latest `daemon-v*` release. Never reuse one.
2. Build from the merged commit on `main`:

   ```powershell
   .\scripts\package-windows.ps1 -Version 0.2.1
   ```

   This writes `dist\windows\sparstrowgen-setup.exe`, its `.sha256`, and the signed
   `sparstrowgen-update.json` with `.sig`, which names this release's installer URL.
3. Publish all four files on one release, marked latest:

   ```powershell
   gh release create daemon-v0.2.1 dist/windows/sparstrowgen-setup.exe dist/windows/sparstrowgen-setup.exe.sha256 dist/windows/sparstrowgen-update.json dist/windows/sparstrowgen-update.json.sig --target <commit> --title "sparstrowgen for Windows 0.2.1" --latest
   ```

4. Check what computers will see: download
   `releases/latest/download/sparstrowgen-update.json` and `sparstrowgen-setup.exe`, and confirm the
   manifest's `version` and `sha256` match the build.
5. A connected computer with automatic updates on installs it within the hour, once no agent work is
   running. **Check now** in Settings → Updates shows it at once.

## Never

- Publish a manifest on a release other than the one holding the installer it names.
- Mark a test or pre-release build as latest: every installed computer would install it.
- Raise `MinDaemonProtocol` before a release those computers can update to has been published.
