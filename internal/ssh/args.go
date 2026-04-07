// Package ssh — args.go parses SSH argv to extract the host alias.
package ssh

import "strings"

// valueFlags is the set of SSH flags that consume the next argument as their value.
// A host-like argument immediately following one of these must not be treated as the host.
var valueFlags = map[string]bool{
	"-b": true,
	"-c": true,
	"-D": true,
	"-E": true,
	"-e": true,
	"-F": true,
	"-I": true,
	"-i": true,
	"-J": true,
	"-L": true,
	"-l": true,
	"-m": true,
	"-o": true,
	"-p": true,
	"-Q": true,
	"-R": true,
	"-S": true,
	"-W": true,
	"-w": true,
}

// ParsedArgs holds the result of parsing raw SSH arguments.
type ParsedArgs struct {
	Alias     string   // the host alias (stripped of user@ prefix)
	User      string   // user@ prefix if present, empty otherwise
	Remaining []string // all other args, order preserved
}

// ParseArgs walks argv (os.Args[1:]) and extracts the host alias.
// SSH flags that take a value argument are handled correctly so their
// values are not mistaken for the host.
func ParseArgs(argv []string) ParsedArgs {
	result := ParsedArgs{
		Remaining: []string{},
	}

	hostFound := false

	for i := 0; i < len(argv); i++ {
		arg := argv[i]

		if hostFound {
			// Everything after the host is a remote command / argument.
			result.Remaining = append(result.Remaining, arg)
			continue
		}

		if valueFlags[arg] {
			// This flag takes the next arg as its value — keep both in Remaining.
			result.Remaining = append(result.Remaining, arg)
			if i+1 < len(argv) {
				i++
				result.Remaining = append(result.Remaining, argv[i])
			}
			continue
		}

		if strings.HasPrefix(arg, "-") {
			// Check for combined form: -p22, -lroot, -i~/.ssh/key, etc.
			if len(arg) > 2 && valueFlags[arg[:2]] {
				result.Remaining = append(result.Remaining, arg)
				continue
			}
			// Boolean / compound flag — keep in Remaining, not the host.
			result.Remaining = append(result.Remaining, arg)
			continue
		}

		// First non-flag argument is [user@]host.
		hostFound = true
		if idx := strings.Index(arg, "@"); idx != -1 {
			result.User = arg[:idx]
			result.Alias = arg[idx+1:]
		} else {
			result.Alias = arg
		}
	}

	return result
}
