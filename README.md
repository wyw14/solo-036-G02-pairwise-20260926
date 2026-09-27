PigFlow is a local HTTP control service for long-distance pipeline pigging operations.

## Pig registry and compatibility

Operators maintain a ledger of pig specifications (dimensions, seal/guide
components, allowed pressure) and evaluate them against route restrictions
(minimum bore and pressure limit along the line):

- `POST /api/pigs` — register a pig spec with its components
- `GET /api/pigs`, `GET /api/pigs/{pigID}` — list/inspect the ledger
- `POST /api/pigs/{pigID}/components/{componentID}/replace` — replace a part;
  bumps the pig revision and invalidates all earlier compatibility conclusions
- `POST /api/pigs/{pigID}/compatibility` — evaluate against a route:
  `{"route_id": "...", "target_pressure": 10.5}` returns a verdict with
  per-check explanations (seal oversize, body clearance, pig pressure rating,
  route pressure limit)
- `GET /api/pigs/{pigID}/compatibility` — stored conclusions; entries whose
  pig revision no longer matches the current configuration are flagged `stale`
