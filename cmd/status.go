package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/joelhelbling/glovebox/internal/digest"
	"github.com/joelhelbling/glovebox/internal/docker"
	"github.com/joelhelbling/glovebox/internal/generator"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/joelhelbling/glovebox/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show profile and Dockerfile status",
	Long:  `Show the current status of your glovebox profiles, images, and Dockerfiles.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Check global profile
	globalProfile, err := profile.LoadGlobal()
	if err != nil {
		return fmt.Errorf("checking global profile: %w", err)
	}

	// Check project profile
	projectProfile, err := profile.LoadProject(cwd)
	if err != nil {
		return fmt.Errorf("checking project profile: %w", err)
	}

	// Build sections
	var sections []ui.StatusSection

	// Base image section
	sections = append(sections, buildBaseSection(globalProfile))

	// Project image section
	sections = append(sections, buildProjectSection(projectProfile, globalProfile))

	// Container section
	sections = append(sections, buildContainerSection(cwd))

	// Render
	status := ui.NewStatus()
	status.Print(sections)
	fmt.Println()

	return nil
}

func buildBaseSection(globalProfile *profile.Profile) ui.StatusSection {
	section := ui.StatusSection{Title: "Base Image"}

	if globalProfile == nil {
		section.Items = append(section.Items,
			ui.StatusItem{Label: "Profile", Value: "Not configured", Status: ui.StatusWarning},
			ui.StatusItem{Value: "Run 'glovebox init --global' to create.", Status: ui.StatusInfo},
		)
		return section
	}

	// Image status
	imageStatus := ui.StatusOK
	imageNote := ""
	if !docker.ImageExists("glovebox:base") {
		imageStatus = ui.StatusWarning
		imageNote = "Run 'glovebox build --base' to build."
	}
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Image", Value: "glovebox:base", Status: imageStatus, Note: imageNote},
	)

	// Profile path
	globalPath, _ := profile.GlobalPath()
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Profile", Value: collapsePath(globalPath)},
	)

	// Mods
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Mods", Value: fmt.Sprintf("%d", len(globalProfile.Mods))},
	)
	for _, m := range globalProfile.Mods {
		section.Items = append(section.Items,
			ui.StatusItem{Value: m, IsList: true, Indent: 1},
		)
	}

	// Dockerfile status
	dockerfilePath := globalProfile.DockerfilePath()
	section.Items = append(section.Items, getDockerfileStatusItems(globalProfile, dockerfilePath, func(mods []string) (string, error) {
		return generator.GenerateBase(mods)
	})...)

	return section
}

func buildProjectSection(projectProfile *profile.Profile, globalProfile *profile.Profile) ui.StatusSection {
	section := ui.StatusSection{Title: "Project Image"}

	if projectProfile == nil {
		section.Items = append(section.Items,
			ui.StatusItem{Label: "Profile", Value: "None (will use glovebox:base)", Status: ui.StatusInfo},
			ui.StatusItem{Value: "Run 'glovebox init' to create a project-specific profile.", Status: ui.StatusInfo},
		)
		return section
	}

	// Image status
	imageName := projectProfile.ImageName()
	imageStatus := ui.StatusOK
	imageNote := ""
	if !docker.ImageExists(imageName) {
		imageStatus = ui.StatusWarning
		imageNote = "Run 'glovebox build' to build."
	}
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Image", Value: imageName, Status: imageStatus, Note: imageNote},
	)

	// Profile path
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Profile", Value: collapsePath(projectProfile.Path)},
	)

	// Mods
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Mods", Value: fmt.Sprintf("%d", len(projectProfile.Mods))},
	)
	for _, m := range projectProfile.Mods {
		section.Items = append(section.Items,
			ui.StatusItem{Value: m, IsList: true, Indent: 1},
		)
	}

	// Dockerfile status
	dockerfilePath := projectProfile.DockerfilePath()
	var baseMods []string
	if globalProfile != nil {
		baseMods = globalProfile.Mods
	}
	section.Items = append(section.Items, getDockerfileStatusItems(projectProfile, dockerfilePath, func(mods []string) (string, error) {
		return generator.GenerateProject(mods, baseMods)
	})...)

	return section
}

func buildContainerSection(cwd string) ui.StatusSection {
	section := ui.StatusSection{Title: "Container"}

	// Workspace
	absPath, _ := os.Getwd()
	dirName := filepath.Base(absPath)
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Workspace", Value: fmt.Sprintf("%s → /%s", collapsePath(absPath), dirName)},
	)

	// Container name and status
	containerName := docker.ContainerName(cwd)
	section.Items = append(section.Items,
		ui.StatusItem{Label: "Container", Value: containerName},
	)

	if docker.ContainerExists(containerName) {
		if docker.ContainerRunning(containerName) {
			section.Items = append(section.Items,
				ui.StatusItem{Label: "Status", Value: "Running", Status: ui.StatusOK},
			)
		} else {
			section.Items = append(section.Items,
				ui.StatusItem{Label: "Status", Value: "Stopped (will resume on next run)", Status: ui.StatusOK},
			)
			// Check for uncommitted changes
			changes, err := getContainerChanges(containerName)
			if err == nil && len(changes) > 0 {
				section.Items = append(section.Items,
					ui.StatusItem{Label: "Changes", Value: fmt.Sprintf("%d uncommitted", len(changes)), Status: ui.StatusWarning},
				)
			}
		}
	} else {
		section.Items = append(section.Items,
			ui.StatusItem{Label: "Status", Value: "Will be created on first run", Status: ui.StatusInfo},
		)
	}

	return section
}

func getDockerfileStatusItems(p *profile.Profile, dockerfilePath string, generateFunc func([]string) (string, error)) []ui.StatusItem {
	var items []ui.StatusItem

	items = append(items,
		ui.StatusItem{Label: "Dockerfile", Value: collapsePath(dockerfilePath)},
	)

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		items = append(items,
			ui.StatusItem{Value: "Not generated", Status: ui.StatusWarning, Indent: 1},
		)
		return items
	}

	// Check if we have build info
	if p.Build.DockerfileDigest == "" {
		items = append(items,
			ui.StatusItem{Value: "Exists but not tracked", Status: ui.StatusWarning, Indent: 1},
		)
		return items
	}

	// Compare digests
	currentDigest, err := digest.CalculateFile(dockerfilePath)
	if err != nil {
		items = append(items,
			ui.StatusItem{Value: fmt.Sprintf("Error reading (%v)", err), Status: ui.StatusWarning, Indent: 1},
		)
		return items
	}

	if currentDigest == p.Build.DockerfileDigest {
		items = append(items,
			ui.StatusItem{Value: "Up to date", Status: ui.StatusOK, Indent: 1},
		)
		items = append(items,
			ui.StatusItem{Value: fmt.Sprintf("Last built: %s", p.Build.LastBuiltAt.Local().Format("2006-01-02 15:04:05 MST")), Status: ui.StatusInfo, Indent: 1},
		)
	} else {
		items = append(items,
			ui.StatusItem{Value: "Modified since generation", Status: ui.StatusWarning, Indent: 1},
		)
	}

	// Check if profile would generate different content
	expectedContent, err := generateFunc(p.Mods)
	if err != nil {
		return items
	}
	expectedDigest := digest.Calculate(expectedContent)

	if expectedDigest != p.Build.DockerfileDigest {
		items = append(items,
			ui.StatusItem{Value: "Profile has changed since last build", Status: ui.StatusWarning, Indent: 1},
		)
	}

	return items
}

func getContainerChanges(name string) ([]string, error) {
	cmd := exec.Command("docker", "diff", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var changes []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			changes = append(changes, line)
		}
	}
	return changes, nil
}
