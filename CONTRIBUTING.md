# Contributing

Thank you for your interest in OpenXOps.

## Licence

OpenXOps is licensed under **FSL-1.1-Apache-2.0** (see the licence line in [README.md](README.md)).
You may use and modify the software freely for your own internal purposes. **Commercial use**
(beyond that scope) requires a separate agreement with the licensor.

## Outside contributions

If you are not the repo owner or an authorized agent working on a dispatched Work Order, we need a
**signed Contributor License Agreement (CLA)** before we can merge your pull request. Open an issue
to request the CLA text and signing instructions.

## How to propose a change

1. Read [AGENTS.md](AGENTS.md) for the project workflow (Work Orders, review, and what may be
   edited in `planning/`).
2. For user-visible behaviour, check whether a Work Order already exists in
   [planning/STATUS.md](planning/STATUS.md).
3. Fork or branch from `main`, run `go run ./scripts/check` (or `task check`), and open a pull
   request with a clear description and test evidence.
4. User-facing command changes should update [docs/user/commands.md](docs/user/commands.md).

## Agents

Automated implementers (Cursor, Antigravity, and similar) follow [AGENTS.md](AGENTS.md): claim
issues before coding, stay inside the Work Order’s *In scope*, and never edit `planning/` unless
the repo owner explicitly asks for a planning fix.
