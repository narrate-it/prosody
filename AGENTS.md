# Prosody agent instructions

## iOS builds

- Use existing approved **self-hosted macOS runners** for iOS builds, archives,
  signing and TestFlight uploads.
- GitHub Actions jobs must use `runs-on: [self-hosted, macOS]`, with any existing
  project-specific runner labels required by the owning runner configuration.
- Do not use `macos-latest` or other GitHub-hosted runners for iOS work. If an
  approved self-hosted runner is unavailable, report the gate without provisioning
  a new runner or falling back to hosted capacity.
- Runner selection does not waive signing, review, capacity or upload gates.
