# Deploying sparstrowgen to Coolify

Only a human can complete this runbook: it requires access to Coolify, DNS and
secrets. It is written against **Coolify v4.3.18** and records what the first
real production deployment did, including the parts where Coolify behaved
differently from its labels.

Follow the seven steps in order. Each step says both **what to do** and **what
becomes true afterwards**, so a failure can be located without guessing.

---

## What you are building

```text
                         Coolify on the VPS
browser ───────────────► web       Next.js UI       app.sparstrow.com
          HTTPS         server    Go API           api.sparstrow.com
                         migrate   database job      exits after succeeding
                         postgres  separate resource

your Windows PC ──────► daemon ──WSS──► server
                         │
                         └── starts the installed Claude, Codex and agy CLIs
```

The web app and API are hosted. The daemon is not: it stays on the computer
where the coding-agent CLIs and project files live, and it dials **out** to the
API. Nothing connects inward to the PC, so no port forwarding or VPN is needed.

### Why there are two domains

- `https://app.sparstrow.com` is what the person opens.
- `https://api.sparstrow.com` receives API and WebSocket traffic.

They are separate origins but the same site (`sparstrow.com`), so the secure
session cookie works between them. The values must match the Coolify domains
exactly; `web.sparstrow.com`, a missing `https://`, or a trailing slash is a
different origin and will be refused.

---

## Before you start

- A Coolify project and environment exist.
- Postgres exists as a **separate resource in the same environment**.
- You can install a GitHub App on the repository.
- You know the VPS public IP.

Create these DNS A records at the DNS provider. The screenshot is from
Hostinger; use the IP of the VPS being deployed to, not an IP copied from this
image.

| Type | Name | Points to |
|---|---|---|
| A | `app` | the Coolify VPS public IP |
| A | `api` | the Coolify VPS public IP |

![Hostinger A records for the app and API subdomains](images/deploy/dns-a-records.png)

DNS can be created before the application. Coolify will check it when the
domains are attached in step 5.

---

## 1. Generate and save the daemon token

The daemon token proves that the program on a computer is allowed to connect to
this server. It is **not** the browser account password.

On Windows PowerShell, generate one with:

```powershell
$bytes = New-Object byte[] 48
$rng = [Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($bytes)
$rng.Dispose()
[Convert]::ToBase64String($bytes)
```

Save the output in a password manager. Do not paste it into chat, source code or
a screenshot. The same value is entered in Coolify in step 4 and on the PC in
step 7. The server rejects values shorter than 32 characters.

**After this step:** one saved secret exists, but nothing is connected yet.

---

## 2. Get the database URL and connect the shared network

Open the Postgres resource in the target Coolify environment and copy its
**internal** connection URL, which has this shape:

```text
postgres://postgres:<password>@<internal-host>:5432/postgres
```

Copy it exactly. Do not construct the hostname from the resource name. On the
first v4.3.18 production deployment, the host in Coolify's internal URL was the
database resource UUID; the visible resource name and container labels were not
a reliable way to predict it.

Now open the sparstrowgen application → **Advanced** → **Docker compose** and
set **Predefined network** to **Connect to predefined network**.

![Coolify Advanced settings with Connect to predefined network selected](images/deploy/predefined-network.png)

### Why this switch matters

Coolify normally gives each Compose application an isolated network. Postgres
is deliberately a separate Coolify resource, so the migration and API
containers cannot resolve or reach it until both resources share Coolify's
predefined network. A wrong setting here produces a real hostname-resolution or
connection error in the `migrate` log; read that error before changing anything.

**After this step:** the application containers will be able to address the
separate Postgres resource using the host already present in its internal URL.

---

## 3. Connect GitHub and create the Compose application

In Coolify, open the project → target environment → **+ New Resource** →
**Application** → **GitHub App**.

If this Coolify instance has no GitHub source yet:

1. Create the GitHub App through Coolify's automated installation.
2. Install it on the `sparstrow` organization.
3. Choose **Only select repositories** and select `sparstrowgen`.
4. Pull-request access can remain **No access** for production-only deployment.
   That disables PR preview/status features; it does not disable deployment
   after a merge. The push webhook on `main` performs that deployment.

Back in the new-resource screen, select `sparstrow/sparstrowgen`, click **Load
Repository**, then use:

| Field | Value |
|---|---|
| Branch | `main` |
| Build Pack | **Docker Compose** |
| Base Directory | `/` |
| Docker Compose Location | `/docker-compose.yaml` |

![Coolify repository screen configured to use the production Compose file](images/deploy/docker-compose-source.png)

Click **Continue**, then **Load Compose** on the application page. Do not paste
Compose YAML into the raw editor and do not use `compose.dev.yaml`; that file is
only for a local Postgres development environment.

### Why Docker Compose

This repository is one product made of three coordinated services:

- `migrate` updates the database and must finish successfully first;
- `server` starts the API only after migration succeeds;
- `web` starts the browser UI alongside the server.

Docker Compose gives Coolify that relationship from the repository instead of
requiring three separately configured applications.

**After this step:** Coolify knows what to build from `main`, but the generated
required variables are not safe to deploy until step 4.

---

## 4. Enter the four environment variables

Open **Environment Variables**. Coolify creates the four required rows from
`docker-compose.yaml`. Open each row using its gear icon and set it exactly as
shown below.

| Variable | Value | Literal | Build time | Runtime |
|---|---|---:|---:|---:|
| `DATABASE_URL` | internal URL copied in step 2 | Yes | Off | On |
| `DAEMON_TOKEN` | saved token from step 1 | Yes | Off | On |
| `WEB_ORIGIN` | `https://app.sparstrow.com` | No | Off | On |
| `API_ORIGIN` | `https://api.sparstrow.com` | No | **On** | **On** |

`Literal` means Coolify keeps `$` characters unchanged. It is important for a
database password or token that happens to contain one. No password hash is
needed: the first browser account is created in step 6.

### Build time versus runtime

- **Build time** supplies a value while an image is being created. The browser
  API address is compiled into the Next.js bundle, so changing `API_ORIGIN`
  requires a rebuild.
- **Runtime** supplies a value while Compose evaluates and starts the deployed
  services. The server needs its database URL, daemon token and allowed browser
  origin then.

Although `API_ORIGIN` is used as a build argument, Coolify v4.3.18 also needs
its Runtime switch on. Coolify runs `docker compose pull` with the runtime
`.env` first, and Compose interpolates the build argument during that command.
With Runtime off, deployment fails with `API_ORIGIN is missing a value` before
the build begins. Keeping Runtime on does not place it in the running web
container; the Compose file only maps it to a build argument.

### Required-variable trap in v4.3.18

The Compose file intentionally uses message-free `${VAR:?}` expressions.
v4.3.18 treated text in `${VAR:?helpful message}` as the generated variable's
actual value. During the first attempt, `migrate` received `set this to the
Postgres resource internal URL` and Goose correctly rejected it as malformed.
Always inspect generated values before deploying; **Load Compose preserves a
bad value already saved in Coolify**.

**After this step:** Compose can be evaluated, the server has its runtime
configuration, and the web image knows the public API address.

---

## 5. Attach the two domains and deploy

Open **Domains**. Replace Coolify's generated `sslip.io` addresses with:

| Service | Protocol | Domain | Internal port | Path |
|---|---|---|---:|---|
| `web` | HTTPS | `app.sparstrow.com` | `3000` | blank |
| `server` | HTTPS | `api.sparstrow.com` | `8080` | blank |

Enable HTTP → HTTPS redirection. Click **Check All DNS**; both rows should say
**DNS matches**.

The Domains table must retain internal ports `3000` and `8080`. The production
Compose file declares them with `expose:` so Coolify's proxy knows where to send
traffic. They are private container ports—do not add `:3000` or `:8080` to the
public URLs.

If Coolify says:

```text
No internal ports. Set port expose or per domain internal port so the proxy can route to this domain.
```

reload the current `/docker-compose.yaml` and confirm the deployed commit
contains the `expose:` entries. Typing the port repeatedly into a domain modal
did not solve this on the first deployment; correcting Compose did.

Click **Deploy**. The expected lifecycle is:

1. `migrate` runs migrations and exits successfully;
2. `server` starts and listens on `0.0.0.0:8080`;
3. `web` starts and serves the UI on port `3000`.

Check the API independently:

```powershell
Invoke-RestMethod https://api.sparstrow.com/api/health
```

An `ok` value of `true` proves the public proxy reaches the API. Also open
`https://app.sparstrow.com`; a fresh database shows **Create your account**.

Coolify may label the Compose resource **Running (no healthcheck)** even though
the `server` service declares a Docker healthcheck. Treat the public health
endpoint and the service logs as the operational evidence, not that aggregate
label.

**After this step:** the website and API are publicly reachable over HTTPS, but
there is still no owner account and no computer connected.

---

## 6. Create the owner account

Nobody has an account on a fresh deployment. Open **Runtime Logs** in Coolify.
The three rows are the three Compose services:

![Coolify Runtime Logs showing migrate, server and web](images/deploy/runtime-services.png)

Expand the newest `server-...` row—the middle row in the screenshot—and find:

```text
this app has no account yet
open it in a browser and use this setup code to create one  setup_code=JINWO3NO...
the code changes every time this server restarts; use the newest one
```

Copy only the value after `setup_code=`. Treat it as a temporary credential:
do not post it in chat or put it in documentation.

Open `https://app.sparstrow.com` and enter:

- the newest setup code;
- the email address you want to use as the login;
- a new password of at least 12 characters, saved in a password manager.

![The first-account form shown by a fresh deployment](images/deploy/create-account.png)

Click **Create account**. This is the deployment's only owner account. Sign-up
closes permanently as soon as it exists, and the setup code is no longer useful.
If the server restarted before submission, retrieve the newest code.

Changing the password later is done from the account menu at the top of the
conversation list. It signs every other browser out; it does not require a
Coolify edit or redeploy.

After sign-in, the conversation screen can still say **Your machine is
unreachable**. That is success for this step: browser authentication works, but
the separate daemon connection in step 7 has not happened yet.

![Signed-in application before the local daemon is connected](images/deploy/signed-in-daemon-offline.png)

**After this step:** the owner can sign in, and nobody else can claim the
deployment through the setup form.

---

## 7. Point the local daemon at production

On the Windows computer that has the agent CLIs and project files, set:

```powershell
setx SERVER_WS "wss://api.sparstrow.com/daemon"
setx DAEMON_TOKEN "<the same saved token from step 1>"
```

Close that terminal and open a **new** one—`setx` changes future processes, not
the shell already open. Start the daemon from the repository:

```powershell
Set-Location D:\sparstrowgen\server
go run ./cmd/daemon
```

The daemon dials outward and logs `connected` on success. In the browser, the
unreachable banner disappears and the installed providers become available in
the provider picker. Confirm all expected providers (`claude`, `codex`, `agy`)
appear before sending a real prompt.

If the daemon says:

```text
the server rejected this machine: DAEMON_TOKEN does not match the server's
```

the Coolify and Windows values differ. Correct the value; do not diagnose that
message as a DNS or firewall problem.

**After this step:** the hosted UI can send work through the hosted server to
the coding-agent CLIs on this computer, while the computer still accepts no
inbound connection.

---

## When something is wrong

| Symptom | What the observed evidence means |
|---|---|
| Deploy is blocked naming a variable | That required variable is empty. Set it under Environment Variables. |
| `API_ORIGIN is missing a value` during `docker compose pull` | Enable both Build time and Runtime for `API_ORIGIN`. |
| Goose cannot parse `set this to the Postgres resource internal URL` | Coolify retained the old generated placeholder as `DATABASE_URL`; replace it with the internal Postgres URL. |
| `migrate` reports hostname resolution or connection failure | Read the full log, then verify the predefined network and the unmodified internal URL from the Postgres resource. |
| Domains shows a red no-internal-port warning | Reload current Compose and verify `expose: 3000` / `8080`; do not keep retyping a port that does not persist. |
| App loads but its requests fail | Confirm the browser is at exactly `https://app.sparstrow.com` and `WEB_ORIGIN` matches it. |
| Sign-in succeeds, then the next request says signed out | Verify HTTPS and `SESSION_SECURE=true`; the `__Host-` cookie requires both. |
| The setup code is rejected | The server restarted after the code was copied. Use the newest `server` runtime log. |
| Sign-up appears even though an account already exists | The server may be unable to read Postgres. Inspect its actual log before changing configuration. |
| Signed-in screen says the machine is unreachable | Hosting and login work; the daemon in step 7 is not connected. |
| Providers remain unavailable | Read the daemon log and verify step 7's WebSocket URL and token. |

## What this deployment does not have yet

- **One account, with no invite or password-reset flow.** Losing the password
  currently requires deliberate database recovery.
- **One shared daemon token for every machine.** A token cannot yet be revoked
  for only one computer ([`Later.md`](../Later.md) L-16).
- **No directory allowlist.** The daemon can run an agent in a folder named by
  an authenticated request.
- **Backups are Coolify's Postgres backups.** There is no application-level
  backup system.
