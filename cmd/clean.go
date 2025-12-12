package cmd

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	cleanBase  bool
	cleanAll   bool
	cleanForce bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean [directory]",
	Short: "Remove glovebox Docker images and volumes",
	Long: `Remove glovebox Docker images and home volumes.

By default, cleans only the current project (or specified directory):
  - Removes the project's Docker image
  - Removes the project's home volume

With --base, also removes the base image (requires confirmation):
  - Everything above, plus glovebox:base

With --all, removes everything glovebox-related (requires confirmation):
  - All glovebox:* images
  - All glovebox-*-home volumes

Use --force to skip confirmation prompts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanBase, "base", false, "Also remove the base image (requires confirmation)")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all glovebox images and volumes (requires confirmation)")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip confirmation prompts")
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	// Check for running containers first
	runningContainers, err := findRunningGloveboxContainers()
	if err != nil {
		return fmt.Errorf("checking for running containers: %w", err)
	}
	if len(runningContainers) > 0 {
		red.Println("Cannot clean while glovebox containers are running:")
		for _, c := range runningContainers {
			fmt.Printf("  - %s (image: %s)\n", c.name, c.image)
		}
		fmt.Println("\nPlease exit the running container(s) first.")
		return fmt.Errorf("running containers detected")
	}

	if cleanAll {
		return cleanAllGlovebox(yellow, green, red)
	}

	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Calculate image and volume names
	hash := sha256.Sum256([]byte(absPath))
	shortHash := fmt.Sprintf("%x", hash)[:7]
	dirName := filepath.Base(absPath)
	imageName := fmt.Sprintf("glovebox:%s-%s", dirName, shortHash)
	volumeName := fmt.Sprintf("glovebox-%s-%s-home", dirName, shortHash)

	// Check if there's anything to clean
	imageFound := imageExists(imageName)
	volumeFound := volumeExists(volumeName)

	if !imageFound && !volumeFound {
		yellow.Printf("No glovebox resources found for %s\n", collapsePath(absPath))
		if cleanBase {
			// Still try to clean base if requested
			return cleanBaseImage(yellow, green, red)
		}
		return nil
	}

	// Clean project resources
	fmt.Printf("Cleaning glovebox resources for %s\n", collapsePath(absPath))

	if imageFound {
		if err := removeImage(imageName, green); err != nil {
			yellow.Printf("Warning: could not remove image %s: %v\n", imageName, err)
		}
	}

	if volumeFound {
		if err := removeVolume(volumeName, green); err != nil {
			yellow.Printf("Warning: could not remove volume %s: %v\n", volumeName, err)
		}
	}

	if cleanBase {
		fmt.Println()
		return cleanBaseImage(yellow, green, red)
	}

	return nil
}

type containerInfo struct {
	name  string
	image string
}

func findRunningGloveboxContainers() ([]containerInfo, error) {
	// Find running containers using glovebox images
	cmd := exec.Command("docker", "ps", "--filter", "ancestor=glovebox", "--format", "{{.Names}}\t{{.Image}}")
	output, err := cmd.Output()
	if err != nil {
		// Also try filtering by image name pattern
		cmd = exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Image}}")
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}

	var containers []containerInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			image := parts[1]
			// Check if it's a glovebox image
			if strings.HasPrefix(image, "glovebox:") {
				containers = append(containers, containerInfo{
					name:  parts[0],
					image: image,
				})
			}
		}
	}

	return containers, nil
}

func cleanBaseImage(yellow, green, red *color.Color) error {
	if !imageExists("glovebox:base") {
		yellow.Println("Base image glovebox:base not found, nothing to clean.")
		return nil
	}

	if !cleanForce {
		red.Println("Warning: This will remove the base image glovebox:base.")
		fmt.Println("All project images depend on this and will need to be rebuilt.")
		fmt.Print("Continue? [y/N] ")

		if !confirmPrompt() {
			fmt.Println("Aborted.")
			return nil
		}
	}

	return removeImage("glovebox:base", green)
}

func cleanAllGlovebox(yellow, green, red *color.Color) error {
	// Find all glovebox images
	images, err := findGloveboxImages()
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}

	// Find all glovebox volumes
	volumes, err := findGloveboxVolumes()
	if err != nil {
		return fmt.Errorf("listing volumes: %w", err)
	}

	if len(images) == 0 && len(volumes) == 0 {
		yellow.Println("No glovebox resources found.")
		return nil
	}

	if !cleanForce {
		red.Println("Warning: This will remove ALL glovebox images and volumes:")
		if len(images) > 0 {
			fmt.Println("\nImages:")
			for _, img := range images {
				fmt.Printf("  - %s\n", img)
			}
		}
		if len(volumes) > 0 {
			fmt.Println("\nVolumes:")
			for _, vol := range volumes {
				fmt.Printf("  - %s\n", vol)
			}
		}
		fmt.Print("\nContinue? [y/N] ")

		if !confirmPrompt() {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Remove all images
	for _, img := range images {
		if err := removeImage(img, green); err != nil {
			yellow.Printf("Warning: could not remove image %s: %v\n", img, err)
		}
	}

	// Remove all volumes
	for _, vol := range volumes {
		if err := removeVolume(vol, green); err != nil {
			yellow.Printf("Warning: could not remove volume %s: %v\n", vol, err)
		}
	}

	return nil
}

func findGloveboxImages() ([]string, error) {
	cmd := exec.Command("docker", "images", "--filter", "reference=glovebox:*", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var images []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			images = append(images, line)
		}
	}
	return images, nil
}

func findGloveboxVolumes() ([]string, error) {
	cmd := exec.Command("docker", "volume", "ls", "--filter", "name=glovebox-", "--format", "{{.Name}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var volumes []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		// Only include volumes that match the glovebox-*-home pattern
		if line != "" && strings.HasSuffix(line, "-home") {
			volumes = append(volumes, line)
		}
	}
	return volumes, nil
}

func removeImage(name string, green *color.Color) error {
	cmd := exec.Command("docker", "rmi", name)
	if err := cmd.Run(); err != nil {
		return err
	}
	green.Printf("Removed image: %s\n", name)
	return nil
}

func removeVolume(name string, green *color.Color) error {
	cmd := exec.Command("docker", "volume", "rm", name)
	if err := cmd.Run(); err != nil {
		return err
	}
	green.Printf("Removed volume: %s\n", name)
	return nil
}

func confirmPrompt() bool {
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
