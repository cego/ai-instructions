package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type composerJSON struct {
	Require map[string]string `json:"require"`
	Config  struct {
		Platform map[string]string `json:"platform"`
	} `json:"config"`
}

func detectFromComposer(projectRoot string, stack *DetectedStack) error {
	path := filepath.Join(projectRoot, "composer.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var c composerJSON
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}

	// Detect PHP
	if stack.PHP == "" && c.Config.Platform != nil {
		if php, ok := c.Config.Platform["php"]; ok {
			stack.PHP = extractMajorVersion(php)
		}
	}

	// Detect PHP
	if stack.PHP == "" {
		if php, ok := c.Require["php"]; ok {
			stack.PHP = extractMajorVersion(php)
		}
	}

	// Detect Laravel
	if stack.Laravel == "" {
		if v, ok := c.Require["laravel/framework"]; ok {
			stack.Laravel = extractMajorVersion(v)
		}
	}

	return nil
}

func detectFromComposerLock(projectRoot string, stack *DetectedStack) error {
	path := filepath.Join(projectRoot, "composer.lock")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var lock struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return err
	}

	if stack.Laravel == "" {
		for _, pkg := range lock.Packages {
			if pkg.Name == "laravel/framework" {
				stack.Laravel = extractMajorVersion(pkg.Version)
				break
			}
		}
	}
	return nil
}
