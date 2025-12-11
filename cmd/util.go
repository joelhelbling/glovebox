package cmd

import "os"

// collapsePath replaces the user's home directory with ~ for display
func collapsePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	if len(path) > len(home) && path[:len(home)+1] == home+"/" {
		return "~" + path[len(home):]
	}
	return path
}
