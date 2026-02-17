package filemanager

import (
	"os"
	"path/filepath"
)

// VerifyResult contains the results of a verification check.
type VerifyResult struct {
	Stack    string
	OK       bool
	Missing  []string
	Tampered []string
}

// VerifyAll verifies all stacks in the instructions directory against expected hashes.
func VerifyAll(projectDir, instructionsDir string, stacks map[string]StackVerifyInfo) []VerifyResult {
	var results []VerifyResult
	for stackID, info := range stacks {
		result := VerifyStack(projectDir, instructionsDir, stackID, info)
		results = append(results, result)
	}
	return results
}

// StackVerifyInfo contains the expected info for verifying a stack.
type StackVerifyInfo struct {
	Hash       string
	Files      []string
	FileHashes map[string]string
}

// VerifyStack verifies a single stack's files exist and the directory hash matches.
func VerifyStack(projectDir, instructionsDir, stackID string, info StackVerifyInfo) VerifyResult {
	result := VerifyResult{Stack: stackID, OK: true}
	stackDir := filepath.Join(projectDir, instructionsDir, stackID)

	// Check each expected file exists
	for _, f := range info.Files {
		path := filepath.Join(stackDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.Missing = append(result.Missing, f)
			result.OK = false
		}
	}

	// If any files are missing, skip hash check
	if len(result.Missing) > 0 {
		return result
	}

	// Check directory hash
	dirHash, err := HashDir(stackDir)
	if err != nil {
		result.OK = false
		result.Tampered = append(result.Tampered, "(hash computation failed)")
		return result
	}

	if dirHash != info.Hash {
		result.OK = false

		// If we have per-file hashes, identify exactly which files changed
		if len(info.FileHashes) > 0 {
			for _, f := range info.Files {
				expected, hasHash := info.FileHashes[f]
				if !hasHash {
					continue
				}
				actual, hashErr := HashFile(filepath.Join(stackDir, f))
				if hashErr != nil || actual != expected {
					result.Tampered = append(result.Tampered, filepath.Join(instructionsDir, stackID, f))
				}
			}
			// Check for extra files not in the expected list
			entries, _ := os.ReadDir(stackDir)
			expectedSet := make(map[string]bool)
			for _, f := range info.Files {
				expectedSet[f] = true
			}
			for _, e := range entries {
				if !e.IsDir() && !expectedSet[e.Name()] {
					result.Tampered = append(result.Tampered, filepath.Join(instructionsDir, stackID, e.Name())+" (unexpected)")
				}
			}
		} else {
			// Fallback: no per-file hashes, report the stack dir as tampered
			result.Tampered = append(result.Tampered, filepath.Join(instructionsDir, stackID, "(dir hash mismatch)"))
		}
	}

	return result
}

// HashFilesInStack computes per-file hashes for all files in a stack directory.
func HashFilesInStack(stackDir string, files []string) (map[string]string, error) {
	hashes := make(map[string]string, len(files))
	for _, f := range files {
		h, err := HashFile(filepath.Join(stackDir, f))
		if err != nil {
			return nil, err
		}
		hashes[f] = h
	}
	return hashes, nil
}
