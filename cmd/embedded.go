package cmd

import "embed"

//go:embed docker/*
var embeddedComposeFiles embed.FS

//go:embed templates/config.yaml
var configTemplate string

//go:embed templates/env.local
var envLocalTemplate string
