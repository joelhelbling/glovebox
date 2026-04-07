package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/spf13/cobra"
)

var (
	cleanImage bool
	cleanAll   bool
	cleanForce bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean [directory]",
	Short: "Remove glovebox container (and optionally image)",
	Long: `Remove glovebox container for the current project.

By default, removes only the container (preserving the image):
  - Discards any uncommitted changes in the container
  - Next run creates a fresh container from the existing image
  - Safe: committed changes in the image are preserved

With --image, also removes the project image:
  - Removes both container and image
  - Next run triggers a full image rebuild
  - Warning: any user-committed changes will be lost

With --all, removes everything glovebox-related (requires confirmation):
  - All glovebox:* images
  - All glovebox-* containers

Use --force to skip confirmation prompts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().BoolVar(&cleanImage, "image", false, "Also remove the project image (loses committed changes)")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all glovebox images and containers (requires confirmation)")
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

	// Calculate image and container names
	imageName := docker.ImageName(targetDir)
	containerName := docker.ContainerName(targetDir)

	// Check if there's anything to clean
	imageFound := rt.ImageExists(imageName)
	containerFound := rt.ContainerExists(containerName)

	if !containerFound && (!cleanImage || !imageFound) {
		yellow.Printf("No glovebox container found for %s\n", collapsePath(targetDir))
		return nil
	}

	// Clean project resources
	fmt.Printf("Cleaning glovebox resources for %s\n", collapsePath(targetDir))

	// Remove container first (must be done before image)
	if containerFound {
		if err := removeContainer(containerName, green); err != nil {
			yellow.Printf("Warning: could not remove container %s: %v\n", containerName, err)
		}
	}

	// Only remove image if --image flag is set
	if cleanImage && imageFound {
		if err := removeImage(imageName, green); err != nil {
			yellow.Printf("Warning: could not remove image %s: %v\n", imageName, err)
		}
	}

	return nil
}

type containerInfo struct {
	name  string
	image string
}

func findRunningGloveboxContainers() ([]containerInfo, error) {
	containers, err := rt.ListContainers("", false) // running only
	if err != nil {
		return nil, err
	}

	var result []containerInfo
	for _, c := range containers {
		if strings.HasPrefix(c.Image, "glovebox:") {
			result = append(result, containerInfo{name: c.Name, image: c.Image})
		}
	}
	return result, nil
}

func cleanAllGlovebox(yellow, green, red *color.Color) error {
	// Find all glovebox images
	images, err := findGloveboxImages()
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}

	// Find all glovebox containers
	containers, err := findGloveboxContainers()
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	if len(images) == 0 && len(containers) == 0 {
		yellow.Println("No glovebox resources found.")
		return nil
	}

	if !cleanForce {
		red.Println("Warning: This will remove ALL glovebox images and containers:")
		if len(containers) > 0 {
			fmt.Println("\nContainers:")
			for _, c := range containers {
				fmt.Printf("  - %s\n", c)
			}
		}
		if len(images) > 0 {
			fmt.Println("\nImages:")
			for _, img := range images {
				fmt.Printf("  - %s\n", img)
			}
		}
		fmt.Print("\nContinue? [y/N] ")

		if !confirmPrompt() {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Remove all containers first (must be done before images)
	for _, c := range containers {
		if err := removeContainer(c, green); err != nil {
			yellow.Printf("Warning: could not remove container %s: %v\n", c, err)
		}
	}

	// Remove all images
	for _, img := range images {
		if err := removeImage(img, green); err != nil {
			yellow.Printf("Warning: could not remove image %s: %v\n", img, err)
		}
	}

	return nil
}

func findGloveboxImages() ([]string, error) {
	return rt.ListImages("glovebox:*")
}

func findGloveboxContainers() ([]string, error) {
	containers, err := rt.ListContainers("glovebox-", true) // all, including stopped
	if err != nil {
		return nil, err
	}

	var names []string
	for _, c := range containers {
		if strings.HasPrefix(c.Name, "glovebox-") {
			names = append(names, c.Name)
		}
	}
	return names, nil
}

func removeContainer(name string, green *color.Color) error {
	if err := rt.ForceRemoveContainer(name); err != nil {
		return err
	}
	green.Printf("Removed container: %s\n", name)
	return nil
}

func removeImage(name string, green *color.Color) error {
	if err := rt.RemoveImage(name); err != nil {
		return err
	}
	green.Printf("Removed image: %s\n", name)
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
