package cmd

import "github.com/fatih/color"

// Package-level color definitions for consistent styling across commands
var (
	colorGreen  = color.New(color.FgGreen)
	colorYellow = color.New(color.FgYellow)
	colorBold   = color.New(color.Bold)
	colorDim    = color.New(color.Faint)
)
