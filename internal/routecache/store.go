package routecache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thereisnotime/sshroute/internal/config"
)

// Defaults for the cache options.
const (
	DefaultTTL         = 7 * 24 * time.Hour    // trust a route for a week
	DefaultLockTimeout = time.Second           // give up on a contended write after this
	lockPollInterval   = 10 * time.Millisecond // how often a contended lock is retried
	stateFileName      = "state.json"          // file name under the state directory
	appDirName         = "sshroute"            // directory under the state root
	lockSuffix         = ".lock"               // sibling lock file suffix
	dirMode            = os.FileMode(0o700)    // state directory permissions
	fileMode           = os.FileMode(0o600)    // state file permissions
	envCache           = "SSHROUTE_CACHE"      // 0/1 override for cache.enabled
	envCachePath       = "SSHROUTE_CACHE_PATH" // override for cache.path
	tempPattern        = ".state-*.json.tmp"   // temp file pattern for atomic writes
	windowsGOOS        = "windows"             // runtime.GOOS on Windows
	localAppDataEnv    = "LOCALAPPDATA"        // Windows per-user local app data
	xdgStateHomeEnv    = "XDG_STATE_HOME"      // XDG state root
	defaultStateSubdir = ".local/state"        // XDG default state root under $HOME
)

// ErrLockTimeout is returned when the state file lock could not be taken in time.
var ErrLockTimeout = errors.New("timed out waiting for state file lock")

// Options is the effective cache configuration after config, env and flags.
type Options struct {
	Enabled     bool
	Path        string
	TTL         time.Duration
	Scope       Scope
	LockTimeout time.Duration
	// Disabled explains why Enabled is false ("--no-cache", "SSHROUTE_CACHE=0", ...).
	Disabled string
}

// DefaultPath returns the default state file path:
//
//  1. $XDG_STATE_HOME/sshroute/state.json when XDG_STATE_HOME is set (any OS)
//  2. %LOCALAPPDATA%\sshroute\state.json on Windows
//  3. ~/.local/state/sshroute/state.json everywhere else, including macOS and Termux
func DefaultPath() (string, error) {
	if xdg := os.Getenv(xdgStateHomeEnv); xdg != "" {
		return filepath.Join(expandTilde(xdg), appDirName, stateFileName), nil
	}
	if runtime.GOOS == windowsGOOS {
		if lad := os.Getenv(localAppDataEnv); lad != "" {
			return filepath.Join(lad, appDirName, stateFileName), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, filepath.FromSlash(defaultStateSubdir), appDirName, stateFileName), nil
}

// ResolveOptions combines the config section, the SSHROUTE_CACHE and
// SSHROUTE_CACHE_PATH environment variables and the --no-cache flag.
// Precedence for on/off: --no-cache, then SSHROUTE_CACHE, then cache.enabled.
// Precedence for the path: SSHROUTE_CACHE_PATH, then cache.path, then DefaultPath.
// getenv is os.Getenv in production.
func ResolveOptions(c config.CacheConfig, noCache bool, getenv func(string) string) (Options, error) {
	o := Options{Enabled: true, TTL: DefaultTTL, Scope: ScopeNetwork, LockTimeout: DefaultLockTimeout}

	if c.Enabled != nil && !*c.Enabled {
		o.Enabled, o.Disabled = false, "cache.enabled is false"
	}
	if v, ok := parseBoolEnv(getenv(envCache)); ok {
		o.Enabled = v
		o.Disabled = ""
		if !v {
			o.Disabled = envCache + "=" + getenv(envCache)
		}
	}
	if noCache {
		o.Enabled, o.Disabled = false, "--no-cache"
	}

	if c.TTL != "" {
		d, err := config.ParseDuration(c.TTL)
		if err != nil {
			return o, fmt.Errorf("cache.ttl: %w", err)
		}
		o.TTL = d
	}
	if c.LockTimeout != "" {
		d, err := config.ParseDuration(c.LockTimeout)
		if err != nil {
			return o, fmt.Errorf("cache.lock_timeout: %w", err)
		}
		o.LockTimeout = d
	}
	if c.Scope != "" {
		o.Scope = Scope(c.Scope)
	}

	switch {
	case getenv(envCachePath) != "":
		o.Path = expandTilde(getenv(envCachePath))
	case c.Path != "":
		o.Path = expandTilde(c.Path)
	default:
		p, err := DefaultPath()
		if err != nil {
			return o, err
		}
		o.Path = p
	}
	return o, nil
}

// parseBoolEnv understands the usual on/off spellings. ok is false for an empty
// or unrecognised value, which leaves the config setting in charge.
func parseBoolEnv(v string) (val, ok bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	}
	return false, false
}

func expandTilde(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[1:])
}

// Store reads and writes one state file.
type Store struct {
	Path        string
	LockTimeout time.Duration
	// Now returns the current time; tests override it.
	Now func() time.Time
}

// NewStore returns a Store for the given options.
func NewStore(o Options) *Store {
	return &Store{Path: o.Path, LockTimeout: o.LockTimeout, Now: time.Now}
}

// Load reads the state without locking. It always returns a usable state; a
// non-nil error means the file was unreadable or corrupt and has been ignored.
func (s *Store) Load() (*State, error) {
	return readFile(s.Path)
}

// Update applies fn to the current state under the file lock and writes the
// result atomically. If the lock is not acquired within LockTimeout it returns
// ErrLockTimeout without writing. A corrupt existing file is replaced; a file
// from a newer format version is left alone and ErrUnsupportedVersion returned.
func (s *Store) Update(fn func(*State)) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("creating state directory %q: %w", dir, err)
	}
	lf, err := os.OpenFile(s.Path+lockSuffix, os.O_CREATE|os.O_RDWR, fileMode) // #nosec G302 G304 -- lock file next to the user's own state file
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}
	defer func() { _ = lf.Close() }()

	if err := acquire(lf, s.LockTimeout); err != nil {
		return err
	}
	defer func() { _ = unlockFile(lf) }()

	st, err := readFile(s.Path)
	if errors.Is(err, ErrUnsupportedVersion) {
		return err
	}
	// Any other read error means a corrupt or unreadable file: start fresh.
	fn(st)
	return writeAtomic(s.Path, st)
}

// Record applies one fallback outcome under the lock.
func (s *Store) Record(alias, network string, o Outcome) error {
	if o.Empty() {
		return nil
	}
	now := s.now()
	return s.Update(func(st *State) { st.Apply(alias, network, o, now) })
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// acquire polls for the exclusive lock until timeout. A zero or negative timeout
// tries exactly once.
func acquire(f *os.File, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		ok, err := tryLockFile(f)
		if err != nil {
			return fmt.Errorf("locking state file: %w", err)
		}
		if ok {
			return nil
		}
		if !time.Now().Before(deadline) {
			return ErrLockTimeout
		}
		time.Sleep(lockPollInterval)
	}
}

// writeAtomic writes st to a temp file in the same directory and renames it over
// path, so readers only ever see a complete old or new file. It deliberately skips
// fsync: a torn file after a crash is just a corrupt cache, which is ignored.
func writeAtomic(path string, st *State) error {
	data, err := encode(st)
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), tempPattern)
	if err != nil {
		return fmt.Errorf("creating temp state file: %w", err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(fileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting permissions on temp state file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp state file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp state file: %w", err)
	}
	committed = true
	return nil
}
