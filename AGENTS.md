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
