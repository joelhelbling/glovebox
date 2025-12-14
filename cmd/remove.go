package cmd

import (
	"fmt"
	"os"

	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <mod>",
	Aliases: []string{"rm"},
	Short:   "Remove a mod from your profile",
	Long: `Remove a mod from your glovebox profile.

Example:
  glovebox remove ai/opencode
  glovebox rm shells/zsh`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
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

	// Try to resolve the mod ID for removal
	// First try exact match, then try with OS suffix
	resolvedModID := resolveModIDForRemoval(modID, p)

	// Remove mod
	if !p.RemoveMod(resolvedModID) {
		fmt.Printf("Mod '%s' is not in your profile.\n", modID)
		return nil
	}

	// Save profile
	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	colorGreen.Printf("✓ Removed '%s' from profile\n", resolvedModID)
	fmt.Println("\nRun 'glovebox build' to regenerate your Dockerfile.")

	return nil
}

// resolveModIDForRemoval finds the actual mod ID in the profile.
// It first checks for an exact match, then tries OS-specific variants.
func resolveModIDForRemoval(modID string, p *profile.Profile) string {
	// Check if exact match exists in profile
	for _, id := range p.Mods {
		if id == modID {
			return modID
		}
	}

	// Try with OS suffix based on profile's OS
	profileOS := getProfileOS(p)
	if profileOS != "" {
		osVariantID := modID + "-" + profileOS
		for _, id := range p.Mods {
			if id == osVariantID {
				return osVariantID
			}
		}
	}

	// Try all known OS variants (in case profile has a different one)
	for _, osName := range mod.KnownOSNames {
		osVariantID := modID + "-" + osName
		for _, id := range p.Mods {
			if id == osVariantID {
				return osVariantID
			}
		}
	}

	// Return original - will result in "not found" message
	return modID
}
