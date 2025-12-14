package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <mod>",
	Short: "Add a mod to your profile",
	Long: `Add a mod to your glovebox profile.

Run 'glovebox mod list' to see available mods.

To create your own custom mod, run:
  glovebox mod create <name>

Examples:
  glovebox add shells/fish
  glovebox add ai/claude-code
  glovebox add custom/my-tool`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	modID := args[0]

	// Load effective profile
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	p, err := profile.LoadEffective(cwd)
	if err != nil {
		return err
	}

	if p == nil {
		return fmt.Errorf("no profile found. Run 'glovebox init' first")
	}

	// Get the profile's OS
	profileOS := getProfileOS(p)

	// Try to resolve the mod ID, handling base names like "editors/vim" -> "editors/vim-ubuntu"
	resolvedModID, requestedMod, err := resolveModID(modID, profileOS)
	if err != nil {
		return err
	}

	// Check if the mod is compatible with the profile's OS
	if err := checkModOSCompatibility(requestedMod, profileOS); err != nil {
		// Suggest the correct variant
		suggestion := suggestModVariant(modID, p)
		if suggestion != "" {
			return fmt.Errorf("%s\nDid you mean '%s'?", err.Error(), suggestion)
		}
		return err
	}

	// Add mod (use the resolved ID)
	if !p.AddMod(resolvedModID) {
		fmt.Printf("Mod '%s' is already in your profile.\n", resolvedModID)
		return nil
	}

	// Save profile
	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	colorGreen.Printf("✓ Added '%s' to profile\n", resolvedModID)
	fmt.Println("\nRun 'glovebox build' to regenerate your Dockerfile.")

	return nil
}

// getProfileOS returns the OS name from the profile's mods, or empty string if not found
func getProfileOS(p *profile.Profile) string {
	for _, modID := range p.Mods {
		m, err := mod.Load(modID)
		if err != nil {
			continue
		}
		if m.Category == "os" {
			return m.Name
		}
	}
	return ""
}

// checkModOSCompatibility verifies that a mod is compatible with the given OS.
// Returns an error if the mod requires a different OS.
func checkModOSCompatibility(m *mod.Mod, profileOS string) error {
	if profileOS == "" {
		return nil // No OS set, allow anything
	}

	for _, req := range m.Requires {
		// Check if requirement is a known OS that differs from profile's OS
		for _, osName := range mod.KnownOSNames {
			if req == osName && req != profileOS {
				return fmt.Errorf("mod '%s' requires '%s', but your profile uses '%s'", m.Name, req, profileOS)
			}
		}
	}
	return nil
}

// resolveModID attempts to resolve a mod ID, handling base names like "editors/vim" -> "editors/vim-ubuntu".
// Returns the resolved mod ID, the loaded mod, and an error if resolution fails.
func resolveModID(modID string, profileOS string) (string, *mod.Mod, error) {
	// First, try to load the exact mod ID
	m, err := mod.Load(modID)
	if err == nil {
		return modID, m, nil
	}

	// If not found and we have a profile OS, try with OS suffix
	if profileOS != "" {
		osVariantID := modID + "-" + profileOS
		m, err = mod.Load(osVariantID)
		if err == nil {
			return osVariantID, m, nil
		}
	}

	// Check if there are any OS variants available for this base name
	availableOSs := []string{}
	for _, osName := range mod.KnownOSNames {
		candidate := modID + "-" + osName
		if _, err := mod.Load(candidate); err == nil {
			availableOSs = append(availableOSs, osName)
		}
	}

	if len(availableOSs) > 0 {
		if profileOS == "" {
			return "", nil, fmt.Errorf("mod '%s' requires an OS-specific variant.\nAvailable for: %s\nAdd an OS mod to your profile first (e.g., 'glovebox add os/ubuntu')",
				modID, strings.Join(availableOSs, ", "))
		}
		return "", nil, fmt.Errorf("mod '%s' is not available for '%s'.\nAvailable for: %s",
			modID, profileOS, strings.Join(availableOSs, ", "))
	}

	return "", nil, fmt.Errorf("mod '%s' not found. Run 'glovebox mod list' to see available mods", modID)
}

// suggestModVariant suggests an alternative mod if the user requested one for a different OS.
// For example, if user requests "shells/zsh-fedora" but profile uses ubuntu, suggest "shells/zsh-ubuntu".
func suggestModVariant(modID string, p *profile.Profile) string {
	profileOS := getProfileOS(p)
	if profileOS == "" {
		return ""
	}

	// Try to find a variant for the profile's OS
	// Handle cases like "shells/zsh-fedora" -> "shells/zsh-ubuntu"
	for _, osName := range mod.KnownOSNames {
		if osName == profileOS {
			continue
		}
		suffix := "-" + osName
		if strings.HasSuffix(modID, suffix) {
			// Try replacing with profile's OS
			candidate := strings.TrimSuffix(modID, suffix) + "-" + profileOS
			if _, err := mod.Load(candidate); err == nil {
				return candidate
			}
		}
	}

	// Handle case where user just types "zsh" but needs "shells/zsh-ubuntu"
	// First, check if a category-prefixed version exists
	for _, category := range []string{"shells", "editors", "tools", "languages", "ai"} {
		// Try with OS suffix
		candidate := category + "/" + modID + "-" + profileOS
		if _, err := mod.Load(candidate); err == nil {
			return candidate
		}
		// Try without OS suffix
		candidate = category + "/" + modID
		if _, err := mod.Load(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
