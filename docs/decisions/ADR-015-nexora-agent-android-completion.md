# ADR-015: Complete the Android backend inside the existing agent and abstraction

- **Status:** Proposed (awaiting owner approval before implementation)
- **Date:** 2026-09-20
- **Depends on:** ADR-008 (local copy bridge), ADR-010 (logical media model)
- **Scope:** Android only. iPhone, USB, external drives and local disks are untouched.

## Context

The instruction was to build a local `NEXORA Agent` for device access and copying.
Inspecting the repository first (required before any edit) changed what that means.

### What already exists

| Piece the task asks for | What is actually there |
|---|---|
| A local agent on Windows | `server/cmd/copybridge` — a real Windows service (`svc.Handler`), listening on `127.0.0.1:32145` |
| Local-only API | `server/internal/copybridge/server.go` — 16 endpoints + SSE on `/api/transfer/events` |
| Authentication | `NEXORA_COPY_BRIDGE_TOKEN` with a constant-time compare, plus a CORS allowlist |
| A device abstraction layer | `TransferBackend` in `transfer/backend.go`, implemented by three backends |
| A transfer engine | `engine.go` + `worker.go` + `scheduler.go`: retries with backoff, size or SHA-256 verification, explicit phases, `PerDeviceConcurrency` |
| 64-bit file sizes | Already `int64` throughout `TransferJobV2` and `TransferFile` |
| Structured classified errors | `TransferError{Code, Message, Retryable, DeviceLost}` |
| Progress in bytes | `PutOptions.OnProgress(transferred int64)`, streamed to the UI over SSE |

**So the agent is not built from scratch, and the abstraction the task asks for is
already present under a different name.** `DeviceProvider` is `TransferBackend`.
`FilesystemProvider` / `AndroidMTPProvider` / `IOSProvider` are `StorageBackend` /
`AndroidBackend` / `GoIOSBackend`, selected by a `BackendFactory`.

### What is actually wrong

Only Android is incomplete. `android_backend.go` implements four of the nine
`TransferBackend` methods; the other three return "not supported".

And Android does **not** use WPD. There is no WPD code anywhere in the
repository. Android is driven by PowerShell `Shell.Application` through
`NameSpace(17)`, and the copy is `Folder.CopyHere(source, 16)`:

- `CopyHere` is **asynchronous and returns nothing** — no completion signal, no
  progress, no error. Progress is obtained by polling the destination file size
  every 1.2 s (`waitForMTPFile`).
- It takes a **file path** as its source, so `PutStream` first writes the reader
  to a temp file on disk. An 80 GB file means 80 GB of extra writes.

### The instruction's own constraint

The instruction says the Windows Shell API must not be the transfer engine, and
should be used only for future Explorer integration. It also says not to invent
dangerous workarounds, and to record a limitation rather than paper over it.

## Decision

### 1. Do not build a second abstraction

`TransferBackend` already is the provider interface. It is extended with
**optional additions only** — never a changed method signature — because three
working backends implement it.

### 2. Do not build WPD now

WPD would solve the real Android limits, but building it means manual COM
interop in Go (`IPortableDevice`, `IPortableDeviceValues`, `IStream`) with no
maintained Go binding, for a working feature. It stays **planned**, recorded in
`docs/agent/ROADMAP.md` with the five-step plan and its entry gate.

The decisive reason it is safe to defer: the abstraction already isolates it.
Replacing `AndroidBackend` with a WPD backend later touches **no** UI, no API and
no engine.

### 3. Complete the Android backend within Shell COM

| # | Gap | Approach |
|---|---|---|
| 1 | `List` returns "not supported" | Enumerate the current MTP folder and return `RemoteEntry` values |
| 2 | `Delete` returns "not supported" | The delete verb is already used inside `Put` for overwrite; expose it as the method |
| 3 | Space check before a copy | Read the target storage's free space; fail early with `INSUFFICIENT_SPACE` |
| 4 | No device-disconnect detection | Check the device still exists on each poll tick; fail with `DEVICE_DISCONNECTED` |
| 5 | Only the first storage is used | Enumerate storages; write to the named one |
| 6 | Fixed `Start-Sleep 500ms` | Poll for the new folder with a bound instead of sleeping a guess |
| 7 | Arabic free-text errors | Return `TransferError` with a code, mapped to a message at the API edge |
| 8 | No capabilities on `Device` | Add a `Capabilities` value so the UI stops trying what a backend cannot do |
| 9 | No structured logging | Emit the fields the instruction lists, never secrets |

### 4. Accept, and record, what Shell COM cannot do

Three Android gaps are **not fixable** through Shell COM, and are not attempted:

| Gap | Why it cannot be fixed | Needs |
|---|---|---|
| Stream without a temp file | `CopyHere` requires a real file path | WPD `IStream` |
| Resume a transfer | `CopyHere` has no append/offset interface | WPD |
| Reliable rename | Shell COM offers no atomic rename over MTP | WPD |

This is the "Problem / Root Cause / Current Limitation / Recommended Solution /
Future Solution" record the instruction asks for. It lives in
`docs/agent/ANDROID.md` §2 and `TRANSFER_ENGINE.md` §8.

### 5. Preserve everything else, verifiably

`go_ios_backend.go`, `storage_backend.go`, `removable_windows.go`, `service.go`,
`engine.go`, `worker.go`, `scheduler.go`, and all of `internal/copybridge` are
**not touched**. `service.go` is specifically not removed: it is reachable from
three callers (the central server, the agent, and the test tool), and two of its
`switch` branches deliberately route a Windows MTP placeholder that is not a real
iOS device to the Android backend — behaviour covered by an existing test.

### 6. No command-execution surface

The agent's API stays capability-based: one endpoint per NEXORA operation. No
`POST /execute`, no arbitrary command, no shell exposed to the UI.

## Consequences

**Positive**

- Android gains browsing, deletion, a space check and disconnect detection — the
  operations a user notices as missing — without touching a working path.
- At most two files change: `android_backend.go` and its test.
- WPD becomes a drop-in replacement later rather than a rewrite.
- The real limits of Shell COM are written down instead of being discovered
  again by the next person.

**Negative / accepted trade-offs**

- Android still cannot stream without a temp file. An 80 GB copy still writes
  the file twice. Recorded, not hidden.
- Progress is still polled at 1.2 s rather than measured, because `CopyHere`
  reports nothing. Recorded.
- `storages[]` on `Device` is added additively; the flat `FreeSpace` and
  `TotalSpace` fields stay so the current UI keeps working.
- Verification of the Android paths is limited: no Android device is attached to
  the development machine, so those tests are written and marked pending rather
  than claimed to pass.

## Alternatives considered

1. **Build WPD now and replace Shell COM.** Rejected for this round: it is the
   largest risk to a working feature, for no user-visible gain that the
   completion work does not already deliver. Deferred behind a recorded gate.
2. **Rename `TransferBackend` to `DeviceProvider`.** Rejected: a rename is a
   breaking change across three backends, the engine and the tests, for a name.
   The abstraction is what matters, and it exists.
3. **Delete `service.go` as legacy code.** Rejected: three callers depend on it,
   and it is the public facade over the v2 engine, not a stale copy.
4. **Hide the three Shell COM limits behind a retry loop.** Rejected: retrying
   cannot add an interface that the API does not have.
5. **Add an ADB provider now.** Rejected: the instruction says not to unless it
   already exists, and it does not. Recorded as planned.

## Verification plan

- Regression: `go build ./...`, `go vet ./...`, `go test ./internal/transfer/... -short`,
  and `go test ./internal/copybridge/...` must pass unchanged.
- Unit: path splitting, PowerShell escaping, device-name extraction, error
  classification — all runnable with no device attached.
- Live Android: discovery, storage listing, browsing, transfer, reconnect and
  mid-transfer disconnect — **recorded as pending**, because no Android device is
  attached to this machine.
- Security: no command surface added; the token and CORS behaviour unchanged.
