package detect

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func extractMajorVersion(version string) string {
	if version == "" {
		return ""
	}

	// Remove common version prefixes and constraints
	version = strings.TrimSpace(version)
	version = strings.Split(version, "||")[0]
	version = strings.Split(version, " ")[0]
	version = strings.TrimLeft(version, "^~><>=v ")

	// Extract only the major version (first numeric segment)
	var major strings.Builder
	for _, r := range version {
		if r >= '0' && r <= '9' {
			major.WriteRune(r)
		} else if r == '.' {
			// Stop at the first dot
			break
		} else {
			// Stop at any other non-numeric character
			break
		}
	}

	return major.String()
}

// DetectStack is used to detect the stack of a project (recursively)
func DetectStack(projectRoot string) (*DetectedStack, error) {
	stack := &DetectedStack{}

	// First: try the root, so root gets to "win"
	if err := detectFromComposer(projectRoot, stack); err != nil {
		return nil, err
	}
	if err := detectFromPackageJson(projectRoot, stack); err != nil {
		return nil, err
	}
	if err := detectFromPackageLockJson(projectRoot, stack); err != nil {
		return nil, err
	}

	ignoredDirs := map[string]bool{
		"node_modules": true,
		"composer":     true,
		"vendor":       true,
	}

	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// if there's a random permission error somewhere, just skip it
			return nil
		}

		// spring root selv over – den er allerede kørt
		if path == projectRoot {
			return nil
		}

		if d.IsDir() {
			name := d.Name()

			// skip dot-folders: .git, .idea, .vscode, ...
			if strings.HasPrefix(name, ".") {
				return fs.SkipDir
			}

			// skip specific folders
			if ignoredDirs[name] {
				return fs.SkipDir
			}

			return nil
		}

		switch d.Name() {
		case "composer.json":
			_ = detectFromComposer(filepath.Dir(path), stack)
		case "composer.lock":
			_ = detectFromComposerLock(filepath.Dir(path), stack)
		case "package.json":
			_ = detectFromPackageJson(filepath.Dir(path), stack)
		case "package-lock.json":
			_ = detectFromPackageLockJson(filepath.Dir(path), stack)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stack, nil
}
