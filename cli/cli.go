package cli

import (
	"fmt"
	"strings"
)

// ParsedArgs is the generic result of parsing a command's arguments. It only
// answers "what did the user provide?" - it has no notion of which flags or
// how many positional arguments are valid for a given command.
type ParsedArgs struct {
	Positional []string
	Flags      map[string]string
}

// Parse splits args into positional arguments and flags. Flags and
// positional arguments may appear in any order.
//
// Recognized flag forms:
//
//	--flag        -> Flags["flag"] = "true"
//	--flag=value  -> Flags["flag"] = "value"
//
// Anything not starting with "--" is treated as a positional argument.
func Parse(args []string) (ParsedArgs, error) {
	parsed := ParsedArgs{
		Positional: []string{},
		Flags:      map[string]string{},
	}

	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			parsed.Positional = append(parsed.Positional, arg)
			continue
		}

		name := arg[2:]

		key, value, hasValue := strings.Cut(name, "=")

		if key == "" {
			return ParsedArgs{}, fmt.Errorf("invalid flag %q: missing flag name", arg)
		}

		if hasValue {
			parsed.Flags[key] = value
		} else {
			parsed.Flags[key] = "true"
		}
	}

	return parsed, nil
}
