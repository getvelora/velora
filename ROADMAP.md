# Velora Roadmap

Velora aims to be a reliable, lightweight, local-first server for movies and TV series. It should be approachable for
first-time self-hosters while supporting high-quality playback, including 4K, HDR, and hardware-accelerated
transcoding.

This roadmap describes direction rather than release guarantees. Priorities may change as real libraries, clients,
and host hardware expose new constraints.

## Product Principles

- Keep media and user data under the owner's control.
- Ship the server, web client, and FFmpeg as one portable container.
- Prefer direct play, then remuxing, then transcoding.
- Detect host and client capabilities instead of assuming them.
- Keep SQLite as the simple default and Postgres as an optional deployment choice.
- Make failures diagnosable and recovery safe.
- Add remote sharing without requiring a Velora-operated cloud service.

## 1. Library Foundation

Delivered foundation:

- Configure libraries through the HTTP API.
- Discover supported video files under configured `/media` paths.
- Persist relative paths, sizes, modification times, and available/missing state.
- Reconcile scans incrementally, preserve inventory after failed walks, and restore missing files without duplicates.
- Trigger synchronous scans and inspect the persisted file inventory through the API.
- Extract normalized container, video, audio, subtitle, codec, and HDR-related details with ffprobe.

Next:

- Complete library create, read, update, and delete workflows.
- Move long scans to resumable background jobs with durable progress and cancellation.
- Detect moved files without losing file identity.
- Group files into movies, series, seasons, and episodes.
- Detect duplicates and surface unmatched or malformed files.
- Show scan progress, errors, history, and manual rescan controls.

## 2. Metadata and Artwork

- Automatically match movies and TV series through TMDB.
- Cache provider responses and downloaded artwork under `/cache`.
- Prefer local NFO files and local artwork when supplied by the owner.
- Provide an interface to correct ambiguous or incorrect matches.
- Support metadata refreshes without losing manual corrections.
- Evaluate TVmaze as a fallback for unresolved TV matches.
- Keep metadata provider credentials configurable and out of the database where practical.

## 3. Direct Playback and Remuxing

- Negotiate playback from browser, codec, container, audio, subtitle, bandwidth, and display capabilities.
- Direct-play compatible source files with byte-range support.
- Remux compatible streams when only the container is unsuitable.
- Support multiple audio and subtitle tracks.
- Persist playback sessions and provide clear reasons for each playback decision.
- Target Safari and Apple devices first, then verify Chrome, Firefox, and Edge.

## 4. Hardware-Accelerated Transcoding

- Transcode incompatible media to adaptive HLS profiles.
- Detect available FFmpeg devices, drivers, decoders, encoders, and filters at startup.
- Support Intel Quick Sync and VAAPI first, NVIDIA NVENC/NVDEC second, and AMD VAAPI third.
- Retain software transcoding as the universal fallback.
- Use hardware decode, scaling, tone mapping, and encode only when the complete pipeline is supported.
- Fall back safely per job when a hardware path fails.
- Provide automatic backend selection with an explicit administrator override.
- Add transcode concurrency, quality, bitrate, and resolution limits.
- Expose a diagnostics page and an optional sample transcode self-test.
- Document device mapping and driver requirements for supported hosts.

## 5. 4K, HDR, and Subtitle Compatibility

- Direct-play 4K and HDR sources whenever the client supports them.
- Preserve bit depth, color primaries, transfer characteristics, and HDR metadata through compatible paths.
- Support HDR10 passthrough and HDR-to-SDR tone mapping on verified hardware pipelines.
- Prevent accidental washed-out output when a color conversion is unavailable.
- Burn image-based or styled subtitles only when the client cannot render them.
- Test common H.264, HEVC Main 10, VP9, and AV1 inputs against supported playback profiles.
- Treat Dolby Vision as a later compatibility project with explicit profile and fallback testing.

## 6. Personal and Household Experience

- Add a local administrator account and household profiles.
- Keep watch history, progress, preferences, and recommendations separate per profile.
- Provide continue watching, recently added, watched state, search, and filtering.
- Add collections, playlists, favorites, and profile-specific playback defaults.
- Support optional OpenSubtitles integration after local and embedded subtitle workflows are stable.
- Add accessible TV-friendly navigation and responsive layouts.

## 7. Reliability and Operations

- Recover cleanly from interrupted scans, metadata refreshes, and transcodes.
- Cancel abandoned transcode jobs and clean partial outputs.
- Track cache entries and enforce configurable quotas and eviction policies.
- Validate migrations and provide documented backup and restore workflows.
- Expose version, readiness, liveness, database, FFmpeg, storage, and hardware status.
- Produce a privacy-conscious diagnostics bundle for troubleshooting.
- Add structured logs, useful metrics, and actionable administrator errors.
- Test upgrades across supported database and configuration versions.

## 8. Setup and Portable Deployment

- Provide a guided first-run setup for storage, libraries, metadata credentials, and hardware detection.
- Keep Docker Compose and the OCI image as the primary supported deployment model.
- Run as a non-root user with configurable UID/GID and predictable file permissions.
- Publish setup and upgrade guides for Docker, Podman, Unraid, TrueNAS SCALE, and Synology where feasible.
- Validate mounts, write permissions, database access, FFmpeg, and hardware devices during setup.
- Maintain a codec, browser, HDR, and hardware acceleration compatibility matrix.
- Keep configuration portable across hosts and avoid platform-specific paths in application data.

Kubernetes is not a planned milestone. Velora's media storage, GPU access, SQLite option, and transcode cache are
single-host concerns, while multiple replicas would require distributed session, job, and cache coordination.
Advanced users may create their own manifests, but official orchestration support should be reconsidered only after
clear demand and a concrete multi-node use case exist.

## 9. Secure Remote Access

- Support secure sessions, device revocation, and invitation management.
- Work correctly behind common HTTPS reverse proxies.
- Publish Tailscale-first guidance for the simplest private remote access.
- Also document direct HTTPS deployment for users who operate their own domain and proxy.
- Apply bandwidth, quality, concurrent-stream, and transcode limits to remote playback.
- Never expose source filesystem paths or grant remote filesystem access.

## 10. Velora Federation

- Pair trusted Velora instances with revocable, short-lived invitations.
- Let owners share selected libraries with selected remote friends.
- Browse shared metadata through the friend's existing Velora interface.
- Stream directly from the owner's instance with owner-controlled limits.
- Keep authentication, authorization, watch state, and audit records local to each instance.
- Handle unavailable peers and changing public endpoints without corrupting local library state.

Federation should not require a central Velora account or cloud service. A hosted discovery, NAT traversal, or relay
service may be evaluated later if direct HTTPS and private-network options prove too difficult for a meaningful number
of users.

## Product Non-Goals

Velora is specifically a movie and TV series server. It will not expand into:

- Music, photo, and general file libraries.
- Full feature parity with established media-server platforms.
- A mandatory cloud account or hosted metadata proxy.

## Deferred Scope

The following are not planned, but may be reconsidered if the core product creates a clear need:

- Native mobile or TV applications.
- A plugin marketplace.
- Multi-node transcoding or horizontally scaled server replicas.
