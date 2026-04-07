// Package ssh — args.go parses SSH argv to extract the host alias.
package ssh

// ParsedArgs holds the result of parsing raw SSH arguments.
type ParsedArgs struct {
	Alias     string   // the host alias (stripped of user@ prefix)
	User      string   // user@ prefix if present, empty otherwise
	Remaining []string // all other args, order preserved
}

// ParseArgs walks argv (os.Args[1:]) and extracts the host alias.
// SSH flags that take a value argument are handled correctly so their
// values are not mistaken for the host.
func ParseArgs(argv []string) ParsedArgs { return ParsedArgs{} } // implemented by A3
