# Deploying to Coolify

Only a human can do this: it needs a dashboard, secrets, and DNS.

**Written against Coolify v4.3.18.** Where the UI has moved, the thing to look for is named.

---

## What you are building

```
                    Coolify (Hostinger VPS)
  browser  ─────►   web      Next.js, one domain
                    server   Go API, its own domain
                    migrate  runs once per deploy, exits
                    postgres  ← already exists, separate resource

  your PC  ─────►   daemon dials OUT to the server over wss://
```

The daemon stays on your machine and dials out. **Nothing ever connects inward to
your PC**, which is why no port forwarding or VPN is involved.

### Two domains, not one

`app.sparstrow.com` for the web app and `api.sparstrow.com` for the API.

This looks like it would break the session cookie, and it does not: `SameSite` is
scoped to the **site** (`sparstrow.com`), not the origin, so the cookie is sent on
requests from the app subdomain to the API subdomain. CORS is already configured
to answer exactly one origin with credentials allowed.

The alternative — one domain with `/api` routed by path — was rejected because it
depends on whether Coolify's proxy strips the path prefix before forwarding, and
getting that wrong produces 404s that look like application bugs.

---

## Before you start

- The repository is connected to Coolify (you have not done this yet).
- The Postgres resource exists in the environment you are deploying to. ✅
- Two A records pointing at the VPS IP: `app.sparstrow.com` and
  `api.sparstrow.com`.

---

## 1. Generate the daemon token

Any long random string. This one is fine:

```bash
openssl rand -base64 48
```

The server rejects anything under 32 characters. You will paste the same value
twice: once into Coolify, once into your machine's environment in step 7.

## 2. Get the database URL

Open the Postgres resource in Coolify → its connection strings are on that page.
Take the **internal** one (not the public one), which looks like:

```
postgres://postgres:<password>@<something>:5432/postgres
```

> **The gotcha that will cost you an hour.** A Docker Compose resource is
> deployed onto its own network, so it cannot see the Postgres resource by
> default. In the application's **Configuration → Advanced**, enable **Connect
> To Predefined Network**.
>
> Enabling it changes how names resolve: Coolify prefixes container names with a
> UUID, so the host in your `DATABASE_URL` must be the **container name shown on
> the Postgres resource page**, not `postgres`. If the deploy fails with a
> hostname that cannot be resolved, this is why.

## 3. Create the application

Coolify → your project → the environment → **+ New Resource** → **Application**
→ your Git source → this repository.

Set:

| Field | Value |
|---|---|
| Build Pack | **Docker Compose** |
| Branch | `main` |
| Docker Compose Location | `/docker-compose.yaml` |
| Base Directory | `/` |

## 4. Set the environment variables

**Configuration → Environment Variables.** All four are required. The compose
file uses `${VAR:?}` so Coolify marks them required and can stop an incomplete
deployment before a container starts.

> **Coolify v4.3.18 gotcha:** do not put a human-readable message after `:?` in
> the compose expression. On the first production deploy, Coolify imported that
> message as the variable's actual value. `migrate` received `set this to the
> Postgres resource internal URL` as `DATABASE_URL` and Goose rejected it as a
> malformed connection string. This repository now uses message-free `${VAR:?}`.
> Still open every generated variable and replace any placeholder text before
> deploying; reloading Compose preserves an existing value.

| Variable | Value |
|---|---|
| `DATABASE_URL` | the internal URL from step 2 |
| `DAEMON_TOKEN` | the random string from step 1 |
| `WEB_ORIGIN` | `https://app.sparstrow.com` |
| `API_ORIGIN` | `https://api.sparstrow.com` |

> Mark `DAEMON_TOKEN` as a secret / build-time-hidden if your Coolify version
> offers it.

`API_ORIGIN` is used at **build** time — it is compiled into the browser bundle.
Changing it later requires a rebuild, not a restart.

## 5. Set the domains and deploy

After the first deploy attempt, Coolify lists the services. Each of `web` and
`server` gets a domain field (the compose file declares `SERVICE_FQDN_WEB_3000`
and `SERVICE_FQDN_SERVER_8080` so they appear):

| Service | Domain |
|---|---|
| `web` | `https://app.sparstrow.com` |
| `server` | `https://api.sparstrow.com` |

**These must match `WEB_ORIGIN` and `API_ORIGIN` exactly** — scheme included, no
trailing slash. A mismatch is not a vague failure: the API refuses the browser's
requests by CORS and refuses its websocket, so the app loads and nothing in it
works.

`docker-compose.yaml` declares the private internal ports (`3000` for `web`,
`8080` for `server`) with Compose `expose:` entries. After loading the current
Compose file, the Domains table must display those numeric ports, not a red
warning triangle. `expose:` is proxy-routing metadata only: it does **not**
publish a host port or add `:3000` / `:8080` to either public URL.

Deploy. Expect, in order: `migrate` runs and exits 0, `server` becomes healthy,
`web` starts. That ordering was verified by running this exact compose file
locally against a real Postgres, so if it does not happen the cause is Coolify's
lifecycle rather than the file — see [`KnownGaps.md`](../KnownGaps.md) G-19.

Coolify requests the TLS certificates automatically once DNS resolves.

### Checking it

```bash
curl https://api.sparstrow.com/api/health
```

`{"ok":true}` means the server is up. It deliberately reports nothing else —
whether your machine is connected is not something an unauthenticated caller
gets told.

## 6. Create your account

Nobody has an account on a fresh deployment, so the app asks you to create one
rather than to sign in.

**Open `server`'s deploy log in Coolify** and find these lines, printed at
startup:

```
this app has no account yet
open it in a browser and use this setup code to create one  setup_code=JINWO3NO...
the code changes every time this server restarts; use the newest one
```

Open `https://app.sparstrow.com`. It shows **Create your account**: paste the
setup code, pick your email and a password of at least 12 characters.

That is the only account this deployment will ever have. **Sign-up closes the
moment it exists** — the endpoint answers "this app already has an account" from
then on, even to the correct setup code — and the screen never comes back.

Why a code at all: what is behind this login runs coding agents on your machine,
so a sign-up form anyone who finds the URL could complete would hand a stranger a
shell. Reading the deploy log is the proof that you own the deployment, and you
are reading it anyway.

If the server has restarted since you looked, use the code from the **newest**
log. The code is held in memory only, so an old one is already dead.

**Changing your password afterwards** is in the app: the account menu at the top
of the conversation list. It signs every other device out and keeps you signed
in where you are. Nothing to edit in Coolify, and no redeploy.

## 7. Point your daemon at it

On your own machine, set these permanently:

```powershell
setx SERVER_WS "wss://api.sparstrow.com/daemon"
setx DAEMON_TOKEN "<the same string from step 1>"
```

Open a **new** terminal (`setx` does not affect the one you ran it in) and start
the daemon. It logs `connected` on success.

If it logs *"the server rejected this machine: DAEMON_TOKEN does not match the
server's"*, the two values differ — that error exists specifically so this does
not look like a network problem.

---

## When something is wrong

| Symptom | Cause |
|---|---|
| Deploy is blocked naming a variable | That required variable is still empty. Set it in Configuration → Environment Variables. |
| Goose says it cannot parse `set this to the Postgres resource internal URL` | Coolify saved the old `${DATABASE_URL:?message}` prompt as the value. Replace `DATABASE_URL` with the Postgres internal URL. |
| `migrate` fails on hostname resolution | Step 2's gotcha — predefined network not enabled, or wrong container name. |
| App loads, everything inside it fails | `WEB_ORIGIN` does not exactly match the web domain. |
| Sign-in succeeds, next request says signed out | `SESSION_SECURE` is not `true`, or the site is not on HTTPS. The `__Host-` cookie prefix requires both. |
| `server` restarts forever after a deploy | Read its log. It names the variable it is unhappy with on the first line. |
| Providers show as unavailable | The daemon is not connected. Check step 7 and your machine. |
| "that setup code is not right" | The server restarted since you copied it. Use the code from the newest log. |
| Sign-up screen when you already have an account | The server cannot reach Postgres, so it cannot tell the account exists. Check `DATABASE_URL`. |

## What this deployment does not have yet

- **One account, and no way to add a second.** There is no invite, no user list
  and no password reset — if you lose the password, recovery is deleting the row
  in Postgres and claiming the app again with a fresh setup code.
- **One shared daemon token for every machine.** It cannot be revoked for one
  machine on its own ([`Later.md`](../Later.md) L-16).
- **No directory allowlist.** The daemon runs an agent in whatever folder a
  request names. Only you can make such a request, so this is depth rather than
  a gate — but it is not built.
- **Backups are Coolify's Postgres backups.** Nothing application-level.
