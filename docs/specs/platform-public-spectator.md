# Public Spectator Contract

## Scope

Public spectator resources provide an anonymous, exported-state-only view of a
logical match. They are independent of the operator read model and never expose
operator artifact locators, delegated URLs, credentials, private snapshots,
event logs, stderr, or admitted bundle bytes.

The field-level HTTP contract is owned by `typespec/namespaces/public/api.tsp`.
This document defines observable behaviour and boundaries only.

## Versioning and transport

The public API family is versioned at `/api/v1-alpha/public/`. Alpha resources
are credential-free, `GET`-only, and return `Access-Control-Allow-Origin: *`.
The alpha contract is explicitly unstable; a breaking change updates this path
and the TypeSpec contract together. A future stable release adds a separate
versioned route family and does not silently redefine the alpha contract.
Payload-local versions (the monotonic state version and replay format version)
do not replace HTTP API versioning.

## Visibility and selected run

The only resource key is `match_id`. A client cannot choose a `run_id`. For a
match with an official completed run, that run is selected; otherwise the
current active/latest run is selected. Promotion changes the selected run
atomically. State versions are monotonically increasing within a selected
`(match_id, run_id)` namespace, and a changed selected run starts a new
namespace.

Queued and leased matches are undiscoverable. Running, persisting, and terminal
matches are discoverable. A state that has not yet been published and every
unreadable replay are represented by the documented unavailable result; backend
errors and private storage details are not observable.

## State and replay boundary

The latest-state resource is built only from atomically published exported
snapshots. Clients may poll it and use its monotonic version and lifecycle to
discard stale responses and stop after a terminal lifecycle.

Terminal replay bytes are served only from a game-produced, versioned public
replay artifact. The platform may store its bounded locator and metadata, but
must not derive a public replay by decoding, filtering, redirecting, or
re-enveloping private artifacts. Payloads larger than 1 MiB, absent, retained,
or unsupported artifacts have the same unavailable result.
