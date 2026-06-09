# Summary

<!-- One or two sentences on the user-visible change and why. -->

## Changes

<!-- Bullet list of notable changes. -->

-

## Verification

<!-- Commands you ran and what you observed. Be specific. -->

- [ ] `cd apps/server && GOCACHE=$PWD/.cache/go-build GOMODCACHE=$PWD/.cache/go-mod go test ./...`
- [ ] `cd apps/web && npm run build`
- [ ] Manually exercised the change via `./velora create` (describe what you clicked / curled)

## Impact

<!-- Tick whichever apply. -->

- [ ] Touches HTTP API surface
- [ ] Touches database schema or migrations (Goose; must work on SQLite and Postgres)
- [ ] Touches `/config` durability or other persistence
- [ ] Touches `./velora`, `compose.dev.yml`, or the public `compose.yml`
- [ ] User-visible UI change (screenshots / clips below)

## Screenshots / clips

<!-- For UI changes. Delete if N/A. -->

## Related issues

<!-- e.g. Closes #123 -->
