# @go/starter

See [API reference](README.md).

## AES-GCM configuration compatibility

Released dependencies select crypto v1.5.9. Shared Symmetric.Decrypt/DecryptBytes authenticate `12-byte IV || ciphertext || 16-byte tag` first, then try the configured legacy IV. Existing APIs remain unchanged. Prefer EncryptBytesAutoIV for new data and upgrade readers before enabling writers. CBC/SM4 formats are unchanged.

## Bounded shutdown (unreleased)

`Register` stop timeouts now bound the entire stop operation, including hooks and
retries, even when `Stop` ignores its context. A non-positive stop timeout uses
`DefaultStopTimeout` (5 seconds). Configure it before `Start` if needed.
`ShutdownTimeout` (15 seconds) bounds all priority tiers together.

Normal shutdown preserves reverse dependency order. A failed service is retried
within its original deadline; after that deadline, remaining dependencies receive
best-effort cleanup. A timed-out goroutine cannot be killed safely by Go and must
not be treated as successfully stopped. The foreground process exits with status
1 on incomplete shutdown or a second SIGINT/SIGTERM, and never reports shutdown
complete for those paths. This manager does not terminate arbitrary child process
groups: applications must own their children and react to parent cancellation.

### Built-in log commands

Applications automatically expose log/logv before service startup, backed by log.RunCommand. `app log` defaults to the configured file with -tail 100 -f; ranges, follow, JSON/raw and explicit files are supported. logv formats stdin or restores files plus their .dict; compressed stdin needs -dict. AddCommand may override these defaults.

### Realtime log integration (unreleased, 2026-11-12)

Released in v1.5.16 with published dependencies and no development replacements. File logs use paired .gz/.dict files; read through logv or embedded commands. Keep both files together; NoCompress retains plaintext.


### Release v1.5.16 — 2026-11-12

Released GitHub dependencies; no local development replacements. Full tests and benchmarks validated with GOWORK=off.

v1.5.17: dependency alignment with safe v1.5.6 / crypto v1.5.11. Development uses explicit local replacements; remove before release. Public API unchanged.

Release v1.5.17: verified with published dependencies and GOWORK=off; all temporary replacements have been removed. Earlier development-validation notes describe historical local checks.
