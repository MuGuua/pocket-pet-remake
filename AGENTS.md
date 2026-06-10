# AGENTS.md

This file defines durable Codex guidance for the `pocket-pet-remake` repository.
It applies to the entire repo unless a deeper `AGENTS.md` overrides it for a subdirectory.

## Project Overview

- This repo is a multiplayer pet game remake with a Godot 4 client in `client/` and a Go backend in `backend/`.
- The client uses autoload singletons such as `App`, `GameState`, `NetClient`, `MessageRouter`, and `HttpClient` declared in `client/project.godot`.
- The backend is organized around transport, protocol, data, and domain modules, with WebSocket message flow under `backend/server/internal/transport/ws/`.
- Current core gameplay domains include auth, world/scene transfer, battle, bag, pet, NPC interaction, and quest progression.

All the code you write needs to have detailed comments, and the default developer is someone who is new to this project.

This game is a mobile game and needs to be developed and adapted according to the characteristics of mobile devices.

## Default Working Style

- Prefer the smallest change that completes the requested behavior without broad refactors.
- Preserve existing naming, scene layout, protocol shape, and gameplay flow unless the user asks for a redesign.
- Before editing, inspect adjacent files and existing patterns in the same feature area; do not introduce a parallel architecture casually.
- When a request spans client and server, verify the full end-to-end contract: command IDs, payload fields, state updates, and push handling.
- If the request is ambiguous, prefer continuing the current architecture instead of inventing a new subsystem.

## Directory Boundaries

- `client/`: Godot scenes, scripts, assets, autoloads, feature controllers, NPC scenes, and world presentation.
- `backend/`: Go services, WebSocket handlers, protocol structs, repositories, migrations, and design docs.
- `docs/` and `backend/docs/`: product/design/protocol notes. Keep them aligned when behavior or contracts change.
- `skills/`: local helper material; do not treat it as runtime game code.

## Godot Client Rules

- Follow existing Godot 4 patterns already used in this repo: autoload-driven orchestration, scene-specific scripts, and feature controllers.
- Prefer editing `.tscn` files through narrowly scoped changes; avoid large mechanical rewrites of scene files.
- Keep gameplay state in `GameState` or the existing owning controller instead of duplicating state in multiple scenes.
- Route network requests through `App`, `NetClient`, `HttpClient`, and `MessageRouter` rather than creating ad hoc networking paths.
- Reuse existing scene-level base scripts when adding new maps, portals, NPCs, or quest interactions.
- When adding UI or interaction logic, ensure it works for the mobile-oriented viewport defined in `client/project.godot`.
- Do not rename autoload singletons, scene paths, or resource paths unless the task explicitly requires it.

## Go Backend Rules

- Keep transport handlers thin. Put gameplay rules in domain services or repositories, not directly in WebSocket handlers.
- Maintain the current layering: protocol/message parsing -> handler orchestration -> domain service -> data repository.
- Reuse existing module boundaries such as `world`, `battle`, and `quest`; avoid cross-module coupling when an event or service boundary already exists.
- When changing payloads, update both protocol structs and the client-side consumers in the same task when possible.
- Preserve backward-compatible field names unless the user explicitly asks for a breaking protocol cleanup.
- Prefer targeted tests near the touched package, especially under `backend/server/internal/...`.

## Protocol And Gameplay Contract Rules

- Treat client-server message schemas as source-of-truth contracts. If one side changes, check the other side immediately.
- Keep command IDs, message names, quest payloads, battle payloads, and world transfer payloads synchronized across both sides.
- For new gameplay flows, confirm these four pieces together: request path, backend authority, push/update path, and client state/UI consumption.
- Do not move authority for battle, quest completion, rewards, or scene transfer to the client.
- When implementing quest or NPC work, prefer the existing documented direction in `backend/docs/quest-system.md`, `backend/docs/quest-protocol.md`, and `docs/market_npc_interaction_design.md`.

## Assets And Generated Content

- Be careful with binary assets and generated map outputs; add only files required for the requested feature.
- Avoid committing accidental editor/system artifacts such as `.DS_Store`.
- Do not mass-regenerate imported or generated assets unless the task specifically requires regeneration.
- If a request involves map assets, preserve the current folder organization under `client/asset/`, `client/scenes/maps/`, and related NPC/script directories.

## Documentation Expectations

- When behavior, protocol, or architecture changes materially, update the closest relevant doc in `docs/` or `backend/docs/`.
- Prefer updating an existing design/protocol doc instead of creating a duplicate note.
- Keep docs concrete: mention message names, scene IDs, command flows, or migration details rather than generic prose.

## Verification Expectations

- For backend-only changes, prefer `go test` on the narrowest affected package first, then widen if needed.
- For protocol or gameplay changes touching multiple backend packages, prefer `go test ./server/...` from `backend/` when feasible.
- For client changes, at minimum sanity-check impacted scene/script paths and confirm signal, autoload, and resource references are still coherent.
- If you cannot run a meaningful verification step, say exactly what was not verified and what the user should run locally.

## Response Style For This Repo

- Be concise and implementation-focused.
- Call out client/server contract risks early.
- Mention file paths when summarizing changes.
- If there are hidden follow-ups, prefer suggesting them as numbered next steps.

## When Unsure

- Choose the more conservative change.
- Prefer consistency with nearby code over abstract best-practice rewrites.
- Ask for clarification only if making the wrong assumption would likely break gameplay flow, protocol compatibility, or project structure.
