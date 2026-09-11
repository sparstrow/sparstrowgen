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

## 1. Generate the password hash

On your own machine:

```bash
make hashpw
```

It asks twice, hides what you type, and prints **two** forms of the same hash.

**Use the second one — the `b64:...` line — in Coolify.** This is not a style
preference. An argon2id hash is full of `$`, Coolify deploys through Docker
Compose, and Compose interpolates `$` in env files: the raw hash arrives at the
container as `=19=19456,t=2,p=1+TsiHSIkx...` with `$argon2id`, `$v` and `$m`
replaced by undefined variables. The server then refuses every password,
correctly, for a reason nowhere near the symptom. This was found by running the
real stack, not by reading it.

The raw `$argon2id$...` form is for a local shell, where nothing eats it. The
server accepts either.

**The password itself is never stored anywhere** — only the hash.

Keep the password somewhere you will find it — there is no reset flow, and
changing it means editing the variable below and redeploying.

## 2. Generate the daemon token

Any long random string. This one is fine:

```bash
openssl rand -base64 48
```

The server rejects anything under 32 characters. You will paste the same value
twice: once into Coolify, once into your machine's environment in step 7.

## 3. Get the database URL

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

## 4. Create the application

Coolify → your project → the environment → **+ New Resource** → **Application**
→ your Git source → this repository.

Set:

| Field | Value |
|---|---|
| Build Pack | **Docker Compose** |
| Branch | `main` |
| Docker Compose Location | `/docker-compose.yaml` |
| Base Directory | `/` |

## 5. Set the environment variables

**Configuration → Environment Variables.** All six are required — the compose
file uses `${VAR:?}`, so a missing one fails the deploy immediately and names
itself, rather than producing a container that restarts forever.

| Variable | Value |
|---|---|
| `DATABASE_URL` | the internal URL from step 3 |
| `OWNER_PASSWORD_HASH` | the **`b64:...`** line from step 1, not the `$argon2id$` one |
| `DAEMON_TOKEN` | the random string from step 2 |
| `WEB_ORIGIN` | `https://app.sparstrow.com` |
| `API_ORIGIN` | `https://api.sparstrow.com` |

> Mark `OWNER_PASSWORD_HASH` and `DAEMON_TOKEN` as secrets / build-time-hidden if
> your Coolify version offers it.

`API_ORIGIN` is used at **build** time — it is compiled into the browser bundle.
Changing it later requires a rebuild, not a restart.

## 6. Set the domains and deploy

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

Then open `https://app.sparstrow.com` and sign in with the password from step 1.

## 7. Point your daemon at it

On your own machine, set these permanently:

```powershell
setx SERVER_WS "wss://api.sparstrow.com/daemon"
setx DAEMON_TOKEN "<the same string from step 2>"
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
| Deploy fails naming a variable | That variable is unset. The compose file is telling you which. |
| `migrate` fails on hostname resolution | Step 3's gotcha — predefined network not enabled, or wrong container name. |
| App loads, everything inside it fails | `WEB_ORIGIN` does not exactly match the web domain. |
| Sign-in succeeds, next request says signed out | `SESSION_SECURE` is not `true`, or the site is not on HTTPS. The `__Host-` cookie prefix requires both. |
| Every password refused, log says the hash cannot be read | The raw `$argon2id$` form was pasted instead of the `b64:` one, and `$`-interpolation ate it. |
| `server` restarts forever after a deploy | Read its log. It names the variable it is unhappy with on the first line. |
| Providers show as unavailable | The daemon is not connected. Check step 7 and your machine. |

## What this deployment does not have yet

- **No way to change the password from the app.** It is `OWNER_PASSWORD_HASH` and
  a redeploy.
- **No directory allowlist.** The daemon runs an agent in whatever folder a
  request names. Only you can make such a request, so this is depth rather than
  a gate — but it is not built.
- **Backups are Coolify's Postgres backups.** Nothing application-level.
