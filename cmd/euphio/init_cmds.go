package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"euphio/internal/assets"
)

var initCmd = &cobra.Command{
	Use:   "init [config_name]",
	Short: "Initialize a new Euphio BBS configuration",
	Long:  "Creates a new configuration file and directory structure for a Euphio BBS, prompting for details.",
	Args:  cobra.MaximumNArgs(1),
	Run:   runInit,
}

type ConfigTemplateData struct {
	BoardName       string
	PrettyBoardName string
	Description     string
	Hostname        string
	Website         string
}

func runInit(cmd *cobra.Command, args []string) {
	configName := "config"
	if len(args) > 0 {
		configName = args[0]
	}

	// Sanitized name for filename and paths
	safeName := sanitizeFilename(configName)

	var data ConfigTemplateData
	data.BoardName = ""
	data.PrettyBoardName = ""
	data.Description = ""
	data.Hostname = ""
	data.Website = ""

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Board Name").
				Value(&data.BoardName),
			huh.NewInput().
				Title("Pretty Board Name").
				Description("Displayed in banners").
				Value(&data.PrettyBoardName),
			huh.NewInput().
				Title("Description").
				Value(&data.Description),
			huh.NewInput().
				Title("Hostname").
				Value(&data.Hostname),
			huh.NewInput().
				Title("Website").
				Value(&data.Website),
		),
	)

	err := form.Run()
	if err != nil {
		log.Fatal(err)
	}

	configFile := safeName + ".yml"
	fmt.Printf("Initializing '%s' (config: %s)...\n", data.BoardName, configFile)

	// Create directory structure
	dirs := []string{"/data", "/keys", "/logs", "/art"}

	for _, dir := range dirs {
		path := safeName + dir
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("Created directory: %s\n", path)
	}

	// Copy views.yml
	menusContent, err := assets.FS.ReadFile("views.yml")
	if err != nil {
		fmt.Printf("Error reading embedded menus template: %v\n", err)
		// Don't exit, just warn, as this might be optional or added later
	} else {
		menusFile := "views.yml"
		if err := os.WriteFile(menusFile, menusContent, 0644); err != nil {
			fmt.Printf("Error writing menus file %s: %v\n", menusFile, err)
		} else {
			fmt.Printf("Created menus file: %s\n", menusFile)
		}
	}

	// Read yml template from assets
	tmplContent, err := assets.FS.ReadFile("config.yml")
	if err != nil {
		fmt.Printf("Error reading embedded config template: %v\n", err)
		os.Exit(1)
	}

	// Replace "config/" with "safeName/" to match the created directories
	tmplContentStr := strings.ReplaceAll(string(tmplContent), "config/", safeName+"/")

	// Parse and execute template
	tmpl, err := template.New("config").Parse(string(tmplContentStr))
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	// Write new config file
	if err := os.WriteFile(configFile, buf.Bytes(), 0644); err != nil {
		fmt.Printf("Error writing config file %s: %v\n", configFile, err)
		os.Exit(1)
	}

	fmt.Printf("Configuration file created: %s\n", configFile)
	fmt.Println("Initialization complete.")
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")
	// Remove non-alphanumeric characters (except underscores and hyphens)
	re := regexp.MustCompile(`[^a-z0-9_-]`)
	name = re.ReplaceAllString(name, "")
	return name
}
