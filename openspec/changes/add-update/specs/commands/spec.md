## ADDED Requirements

### Requirement: update — self-update to the latest release
`sshroute update` SHALL download the latest GitHub release archive for the running platform, verify its sha256 against `checksums.txt`, verify the cosign signature of `checksums.txt` when `cosign` is available, and atomically replace the running executable. `--check` SHALL only report availability, and `--force` SHALL reinstall the latest version even if already current.

#### Scenario: Already on the latest version
- **WHEN** the installed version is the same as (or newer than) the latest release and `--force` is not set
- **THEN** no download occurs and the command reports that it is already up to date

#### Scenario: Check only
- **WHEN** `--check` is passed and a newer release exists
- **THEN** the command prints the current and latest versions and the release URL, and does not modify the binary

#### Scenario: Verified update
- **WHEN** a newer release exists and its downloaded archive's sha256 matches `checksums.txt`
- **THEN** the binary in the archive replaces the running executable

#### Scenario: Checksum mismatch aborts
- **WHEN** the downloaded archive's sha256 does not match `checksums.txt`
- **THEN** the command aborts with an error and the running executable is left unchanged

#### Scenario: Cosign verification enforced when available
- **WHEN** `cosign` is installed and its verification of the `checksums.txt` bundle fails
- **THEN** the update aborts before replacing the binary

#### Scenario: Cosign absent
- **WHEN** `cosign` is not installed
- **THEN** signature verification is skipped with a notice and the sha256 check still gates the update
