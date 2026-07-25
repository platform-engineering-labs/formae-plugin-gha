# Changelog

All notable changes to the formae GitHub Actions plugin are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Install with `sudo formae plugin install gha` on the host that runs the
formae agent.

## [0.1.3]

### Changed

- Secret payload fields are now typed `formae.SecretValue` so their values are hashed at rest end-to-end (previously stored in cleartext on the read/actual-state path). Covers the `value` field on `GHA::Environment::Secret`, `GHA::Org::Secret`, and `GHA::Repo::Secret`. Requires a formae agent on the matching release; `minFormaeVersion` is bumped to 0.88.0.

## [0.1.2]

### Changed

- Now available on the platform.engineering Hub. Install with `sudo formae
  plugin install gha` on the host that runs the formae agent.

## [0.1.0]

### Added

- Initial release of the GitHub Actions plugin as a standalone package built on
  the formae Plugin SDK.
