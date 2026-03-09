# Releasing `gpc`

## Maintainer Checklist

1. Merge the release-ready changes into `main`.
2. Run the local safety checks:

   ```bash
   make dev
   goreleaser check
   goreleaser release --snapshot --clean
   ```

3. Create the release tag:

   ```bash
   git checkout main
   git pull --ff-only origin main
   git tag v0.1.0
   git push origin v0.1.0
   ```

4. Wait for the `Release` GitHub Actions workflow to finish.
5. Verify the GitHub Release contains:
   - one archive for each supported platform
   - one SHA256 checksums file
   - generated release notes
6. Download one artifact and confirm the embedded metadata:

   ```bash
   ./gpc --version
   ```

7. Spot-check the install steps in `README.md`.

## Release Notes

- Tags must use the `vMAJOR.MINOR.PATCH` format.
- `v0.1.0` is the first public release tag.
- Homebrew and a custom install script are intentionally out of scope for this release line.
