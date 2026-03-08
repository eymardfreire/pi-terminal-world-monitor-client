# OpenSpec – AI assistant instructions

This project uses [OpenSpec](https://github.com/Fission-AI/OpenSpec) for spec-driven development.

## Workflow

- **/opsx:new** – Create a new change proposal (proposal, specs, design, tasks).
- **/opsx:ff** – Fast-forward: generate all planning docs for the current change.
- **/opsx:apply** – Implement tasks from the current change.
- **/opsx:archive** – Archive the completed change and update main specs.

## Layout

- `openspec/changes/` – Active change proposals.
- `openspec/archive/` – Archived (completed) changes.
- Each change has: `proposal.md`, `specs/`, `design.md`, `tasks.md`.

Follow the specs and design in the current change when implementing. Update artifacts as the work evolves.

## Deploy: push and restart backend

When you make changes that require updating the backend on the VPS (e.g. new or changed API in `backend/`), **always give the user**:

1. **Commands to run locally** to commit and push, with a **relevant, specific commit message** (describe what changed, e.g. "Add 1h/7d price change to crypto top panel", not "Updates").
2. **Commands to run on the VPS** to pull and restart the backend.

See **docs/HANDOFF-PROGRESS.md** (§ Deploy workflow: push and restart backend) for the exact command templates and VPS paths.
