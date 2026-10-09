# quanly-phongtro — project memory

Shared knowledge for every coding agent (Claude, Codex, Gemini/agy, subagents). `CLAUDE.md` is a symlink to this file. Coding rules template: `AGENTS.example.md`. Never put secrets here.

## Stack & runtime

- Go API (chi + bun, PostgreSQL) + React/Vite frontend (`frontend/`, shadcn base-luma); the frontend is built into `internal/web/dist` and embedded in one binary.
- Production `https://quanly.ptro.site` runs on this machine: systemd `quanly-phongtro-api`, working dir `/home/dell/actions-runner/_work/quanly-phongtro/quanly-phongtro`, logs `journalctl -u quanly-phongtro-api`. DB DSN in `.env` (`POSTGRES_DSN`).
- `.env` and `uploads/` live outside the runner workspace in `~/quanly-phongtro/` and are symlinked in by `scripts/deploy/set-env.sh`: `actions/checkout` can wipe the whole workspace (on 2026-10-08 it wiped all uploaded CCCD/contract images). `uploads/` is rsynced (no `--delete`) to the offsite host by `scripts/backup-db-offsite.sh`. That script uses a dedicated key `~/.ssh/qlpt_backup_ed25519` (passphrase in `~/dotfile/qlpt/backup-ssh.pass`, `restrict` on ocl) loaded into a per-run ssh-agent: the runner service has no agent, so the personal passphrase-protected `id_ed25519` fails there with `Permission denied (publickey)`.
- Ingress: Cloudflare routes Vietnamese ISPs (Viettel) to distant colos (LAX/WAW/MAD...) with heavy packet loss, so a 788KB upload took 30–70s or timed out (measured 2026-10-09). Bypass = an edge server running nginx (TLS + proxy) ← reverse SSH tunnel from this machine (systemd **user** unit `qlpt-edge-tunnel@<ssh-host>`, restricted edge user `qlpt-tunnel`). Deploy/move it with `scripts/deploy/edge/deploy-edge.sh install|cert|status|remove <ssh-host>` (guide: `scripts/deploy/edge/README.md`) (domains → ports in `sites.conf`); it auto-detects an nginx SNI router on :443 (as on `ocl`) and never touches other sites. Edges: `ocl` 149.118.154.51 (Oracle, Johor MY, RTT ~48ms) serves both domains since 2026-10-10 (Vultr Seattle was tried first: RTT ~215ms, removed). The DNS record picks the path: A → edge IP, DNS only; proxied CNAME to the cloudflared tunnel = Cloudflare (rollback).
- Deploy = push to `develop` → `.github/workflows/deploy.yml` on a self-hosted runner (lint/test → build → backup → migrate → restart). Pushing deploys production, so only push when the user asks.
- CI tests: `scripts/deploy/lint-test.sh` (`./internal/{service,handler,repository}/...` and `./pkg/...`).

## Architecture rule: third-party integrations live in `pkg/<name>`

- Each integration is a self-contained, copy-pasteable module in `pkg/<name>/`: Go standard library only, a `guide.md`, an optional `sql/` schema and `web/` TypeScript.
- The app keeps exactly **one** adapter file, `internal/service/<domain>/<name>_adapter.go` (+ test), that bridges the module to app types (provider registry, invoices, DB). No duplicated aliases/types elsewhere; handlers and `cmd/api/main.go` use `<name>.*` directly.
- Frontend imports module web files through a Vite + tsconfig alias `@<name>/*` → `../pkg/<name>/web/*`. The module API client takes a transport; the app passes axios `apiClient` (adds DPoP).
- Done: **SePay** (`pkg/sepay`), **Zalo bot** (`pkg/zalobot`, backend only: API client, webhook parse/secret check, safe image download). Zalo chat commands, reply texts and account/room linking stay in `internal/service/zalo`; app glue is `internal/service/zalo/zalobot_adapter.go`.

## SePay

- Module + integration guide: `pkg/sepay/guide.md`. App glue: `internal/service/payment/sepay_adapter.go`. Feature doc: `Documents/feature/sepay_integration.md`.
- Preferred provider order when sending an invoice: SePay, then PayOS.
- Webhook `id` is an integer but API v2 `id` is a UUID, so the same bank transfer cannot be deduplicated by reference. Guard: if the link is already `PAID`, only write an audit event and never settle twice (`ProcessVerifiedTransaction`).
- HMAC: `X-SePay-Signature: sha256=hex(HMAC_SHA256(secret, "{X-SePay-Timestamp}.{raw_body}"))`, ±5 min.
- E2E testing: skill `.agents/skills/sepay-e2e/` (scripts for login, invoice, Zalo send, simulate, status, reconcile, cleanup). SePay Test Mode only. Credentials live outside the repo in `~/.config/quanly-phongtro/` (chmod 600); never print them, and never extract the webhook secret from the SePay page (the user enters it in Settings → SePay).
- Test invoices use period `2099-xx` on rooms without a Zalo group/linked tenants, and must be cleaned up afterwards (`cleanup.sh`).

## Zalo bot constraints (https://bot.zapps.me/docs/)

- Only 5 webhook events (text/image/sticker/voice/unsupported). There is no "bot added to group" event.
- `chat` = `{id, chat_type}`: no group name and no group-info API, so groups are linked to rooms manually (the bot replies with the Group ID and the manager pastes it into the room).
- In groups the bot only receives messages when it is @mentioned or replied to.

## Gotchas learned

- bun inserts every struct column. For tables whose `id` has a DB default (`uuidv7()`), use `ExcludeColumn("id", ...)` + `Returning("id, ...")`; an empty string in a uuid column fails (`invalid input syntax for type uuid: ""`). Before 2026-10-08 this silently broke every payment link and QR.
- sqlmock tests must assert the exact SQL (`^INSERT INTO ... (cols) VALUES (...)$`), never `.*`; the loose matcher hid the bug above.
- UI: new primitives come from `pnpm dlx shadcn@latest add` (Base UI, not Radix). Icons come from `@/components/icons`. Use colour tokens only; shared components live in `src/components/shared/`.
