//go:build e2e

package e2e

import (
	"fmt"

	"github.com/joelhelbling/glovebox/internal/mod"
)

// ModInfo contains the essential info needed for compatibility checking
type ModInfo struct {
	ID       string
	Name     string
	Provides []string
	Requires []string
	Category string
}

// LoadAllEmbeddedMods loads all embedded mods and returns their info
func LoadAllEmbeddedMods() ([]ModInfo, error) {
	allMods, err := mod.ListAll()
	if err != nil {
		return nil, err
	}

	var result []ModInfo
	for category, ids := range allMods {
		for _, id := range ids {
			m, err := mod.Load(id)
			if err != nil {
				continue // Skip mods that can't be loaded
			}

			info := ModInfo{
				ID:       id,
				Name:     m.Name,
				Provides: m.EffectiveProvides(),
				Requires: m.Requires,
				Category: category,
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// ModsCompatibleWithOS returns all mod IDs that can be installed on the given OS.
// It uses a fixed-point algorithm: starting with what the OS provides, it iteratively
// finds mods whose requirements are satisfied, adding their provides to the available set.
func ModsCompatibleWithOS(osName string) ([]string, error) {
	allMods, err := LoadAllEmbeddedMods()
	if err != nil {
		return nil, err
	}

	// Build a map for quick lookup
	modByID := make(map[string]ModInfo)
	for _, m := range allMods {
		modByID[m.ID] = m
	}

	// Start with what the OS itself provides
	// OS mods provide their name plus typically "base"
	available := make(map[string]bool)
	osModID := "os/" + osName
	if osMod, ok := modByID[osModID]; ok {
		for _, p := range osMod.Provides {
			available[p] = true
		}
	}
	available[osName] = true // OS name is always available

	// Track which mods are compatible
	compatible := make(map[string]bool)
	compatible[osModID] = true // OS mod itself is compatible

	// Fixed-point iteration: keep finding mods until no new ones are found
	changed := true
	for changed {
		changed = false
		for _, m := range allMods {
			if compatible[m.ID] {
				continue // Already marked compatible
			}

			// Skip other OS mods - they can't be "added" to compatibility
			// Only the starting OS is valid
			if m.Category == "os" {
				continue
			}

			// Check if all requirements are satisfied
			allSatisfied := true
			for _, req := range m.Requires {
				if !available[req] {
					allSatisfied = false
					break
				}
			}

			if allSatisfied {
				compatible[m.ID] = true
				// Add what this mod provides to available set
				for _, p := range m.Provides {
					available[p] = true
				}
				changed = true
			}
		}
	}

	// Convert to slice, excluding OS mods (we test those separately)
	var result []string
	for id := range compatible {
		if modByID[id].Category != "os" {
			result = append(result, id)
		}
	}

	return result, nil
}

// LeafModsForOS returns mods that are "interesting" to test - mods that users
// would actually add to their profile (not just dependencies).
// This filters out low-level infrastructure mods like "mise" that are only
// dependencies of other mods.
func LeafModsForOS(osName string) ([]string, error) {
	compatible, err := ModsCompatibleWithOS(osName)
	if err != nil {
		return nil, err
	}

	allMods, err := LoadAllEmbeddedMods()
	if err != nil {
		return nil, err
	}

	// Build provides map to see what's used as a dependency
	usedAsDependency := make(map[string]bool)
	for _, m := range allMods {
		for _, req := range m.Requires {
			usedAsDependency[req] = true
		}
	}

	// Build ID to mod map
	modByID := make(map[string]ModInfo)
	for _, m := range allMods {
		modByID[m.ID] = m
	}

	// Filter to mods that aren't primarily used as dependencies
	// (but include them if they're in user-facing categories)
	userFacingCategories := map[string]bool{
		"shells":    true,
		"editors":   true,
		"languages": true,
		"ai":        true,
	}

	var result []string
	for _, id := range compatible {
		m := modByID[id]
		// Include if it's a user-facing category
		if userFacingCategories[m.Category] {
			result = append(result, id)
			continue
		}
		// Include tools that aren't just dependencies
		if m.Category == "tools" {
			// Check if this mod's name is used as a dependency by others
			isInfrastructure := false
			for _, p := range m.Provides {
				if usedAsDependency[p] {
					isInfrastructure = true
					break
				}
			}
			if !isInfrastructure {
				result = append(result, id)
			}
		}
	}

	return result, nil
}

// CompatibilityMatrix returns a map of OS name to compatible mod IDs
func CompatibilityMatrix() (map[string][]string, error) {
	result := make(map[string][]string)
	for _, osName := range mod.KnownOSNames {
		mods, err := ModsCompatibleWithOS(osName)
		if err != nil {
			return nil, err
		}
		result[osName] = mods
	}
	return result, nil
}

// ResolveDependencies takes a mod ID and OS name, and returns a list of all
// mod IDs needed to satisfy the mod's requirements (including transitive deps).
// It automatically finds OS-compatible mods that provide required capabilities.
func ResolveDependencies(modID, osName string) ([]string, error) {
	allMods, err := LoadAllEmbeddedMods()
	if err != nil {
		return nil, err
	}

	// Build lookup maps
	modByID := make(map[string]ModInfo)
	for _, m := range allMods {
		modByID[m.ID] = m
	}

	// Build a map of what each capability is provided by (for this OS)
	compatibleMods, err := ModsCompatibleWithOS(osName)
	if err != nil {
		return nil, err
	}
	compatibleSet := make(map[string]bool)
	for _, id := range compatibleMods {
		compatibleSet[id] = true
	}

	// Map from provided name -> mod IDs that provide it (filtered to OS-compatible)
	providers := make(map[string][]string)
	for _, m := range allMods {
		if !compatibleSet[m.ID] && m.Category != "os" {
			continue // Skip mods not compatible with this OS
		}
		for _, p := range m.Provides {
			providers[p] = append(providers[p], m.ID)
		}
	}

	// Also add the OS mod's provides
	osModID := "os/" + osName
	if osMod, ok := modByID[osModID]; ok {
		for _, p := range osMod.Provides {
			providers[p] = append(providers[p], osModID)
		}
	}

	// Track what's been resolved
	resolved := make(map[string]bool)
	resolved[osModID] = true // OS is always resolved
	var result []string

	// Recursive resolution function
	var resolve func(id string) error
	resolve = func(id string) error {
		if resolved[id] {
			return nil
		}

		m, ok := modByID[id]
		if !ok {
			return fmt.Errorf("mod not found: %s", id)
		}

		// Resolve dependencies first
		for _, req := range m.Requires {
			// Check if already satisfied
			satisfied := false
			for resolvedID := range resolved {
				rm := modByID[resolvedID]
				for _, p := range rm.Provides {
					if p == req {
						satisfied = true
						break
					}
				}
				if satisfied {
					break
				}
			}
			// Also check OS provides
			if !satisfied {
				if osMod, ok := modByID[osModID]; ok {
					for _, p := range osMod.Provides {
						if p == req {
							satisfied = true
							break
						}
					}
				}
			}

			if satisfied {
				continue
			}

			// Try to find a provider for this requirement
			providerIDs := providers[req]
			if len(providerIDs) == 0 {
				return fmt.Errorf("no provider found for %q (required by %s)", req, id)
			}

			// Use the first compatible provider
			if err := resolve(providerIDs[0]); err != nil {
				return err
			}
		}

		resolved[id] = true
		result = append(result, id)
		return nil
	}

	if err := resolve(modID); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper to load a mod's YAML directly for debugging
func loadModYAML(id string) (*mod.Mod, error) {
	return mod.Load(id)
}
