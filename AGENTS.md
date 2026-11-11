# AGENTS.md

Invoke the `go-tutor` skill for this project.

## What go-tutor does

Guides Go learning through phased plans in `../plans/`. The agent
describes what to write in prose — no code blocks. You write all code.
The agent reviews, explains concepts, and adjusts plans on request. It
does not edit source files, run `go get`, or touch `go.mod`/`go.sum`.

## Plan structure

Phases 0-6 are written. Phase 0 locks architecture decisions. Phases
1-4 build the gateway logic. Phase 5 adds graceful shutdown. Phase 6
adds Dockerization. Later phases (Postgres, auth, Redis) are not yet
written.

See `../README.md` at the project root for the full plan index.
