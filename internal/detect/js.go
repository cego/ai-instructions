package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func detectFromPackageJson(projectRoot string, stack *DetectedStack) error {
	path := filepath.Join(projectRoot, "package.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var p packageJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	get := func(name string) (string, bool) {
		if v, ok := p.Dependencies[name]; ok {
			return v, true
		}
		if v, ok := p.DevDependencies[name]; ok {
			return v, true
		}
		return "", false
	}

	if v, ok := get("nuxt"); ok && stack.Nuxt == "" {
		stack.Nuxt = extractMajorVersion(v)
	}
	if v, ok := get("vue"); ok && stack.Vue == "" {
		stack.Vue = extractMajorVersion(v)
	}
	if v, ok := get("@nuxt/ui"); ok && stack.NuxtUI == "" {
		stack.NuxtUI = extractMajorVersion(v)
	}

	return nil
}

type lockFile struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`

	Packages map[string]struct {
		Version      string            `json:"version"`
		Dependencies map[string]string `json:"dependencies"`
	} `json:"packages"`
}

func detectFromPackageLockJson(projectRoot string, stack *DetectedStack) error {
	path := filepath.Join(projectRoot, "package-lock.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return err
	}

	if stack.NuxtUI == "" {
		if dep, ok := lock.Dependencies["@nuxt/ui"]; ok && dep.Version != "" {
			stack.NuxtUI = extractMajorVersion(dep.Version)
			return nil
		}

		if pkg, ok := lock.Packages["node_modules/@nuxt/ui"]; ok && pkg.Version != "" {
			stack.NuxtUI = extractMajorVersion(pkg.Version)
			return nil
		}
	}

	return nil
}
