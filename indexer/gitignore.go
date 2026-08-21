package indexer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// gitignoreRule is one parsed line from a .gitignore file.
type gitignoreRule struct {
	// anchored patterns (a leading "/", or a "/" anywhere before the end)
	// only match relative to rootDir, per gitignore(5). Unanchored
	// patterns match a name at any depth, the same way IgnoreDirs already
	// does.
	anchored bool
	// dirOnly patterns (a trailing "/") only match directories.
	dirOnly bool
	pattern string
}

// loadGitignore reads the .gitignore directly at rootDir, if present. Only
// the root .gitignore is honored - nested .gitignore files in
// subdirectories, and negation ("!pattern"), are not supported. This is a
// deliberately simplified subset covering the common case (a single
// project-root .gitignore with plain entries), not the full gitignore spec.
func loadGitignore(rootDir string) ([]gitignoreRule, error) {
	f, err := os.Open(filepath.Join(rootDir, ".gitignore"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rules []gitignoreRule

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		dirOnly := strings.HasSuffix(line, "/")
		if dirOnly {
			line = strings.TrimSuffix(line, "/")
		}

		anchored := strings.HasPrefix(line, "/")
		if anchored {
			line = strings.TrimPrefix(line, "/")
		} else if strings.Contains(line, "/") {
			// A "/" anywhere before the end also anchors a pattern to the
			// .gitignore's own directory, per gitignore(5).
			anchored = true
		}

		if line == "" {
			continue
		}

		rules = append(rules, gitignoreRule{anchored: anchored, dirOnly: dirOnly, pattern: line})
	}

	return rules, scanner.Err()
}

// matchesGitignore reports whether path (with isDir reporting whether it's
// a directory) is excluded by rules, loaded from rootDir's .gitignore.
func matchesGitignore(rules []gitignoreRule, rootDir, path string, isDir bool) bool {
	if len(rules) == 0 {
		return false
	}

	name := filepath.Base(path)

	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		rel = name
	}

	for _, rule := range rules {
		if rule.dirOnly && !isDir {
			continue
		}

		candidate := name
		if rule.anchored {
			candidate = rel
		}

		if matched, _ := filepath.Match(rule.pattern, candidate); matched {
			return true
		}
	}

	return false
}
