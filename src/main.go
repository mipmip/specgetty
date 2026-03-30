package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/mipmip/specgetty/src/scanner"
	"github.com/mipmip/specgetty/src/ui"
)

func getDefaultConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "specgetty", "config.yml")
}

//go:embed config.yml
var defaultConfig string

//go:embed VERSION
var version string

func init() {
	version = strings.TrimSpace(version)
}

func main() {
	app := cli.NewApp()
	app.Name = "specgetty"
	app.Version = version
	app.Usage = "Finds OpenSpec projects on your local machine"
	app.EnableBashCompletion = true
	app.CommandNotFound = func(c *cli.Context, cmd string) {
		fmt.Printf("ERROR: Unknown command '%s'\n", cmd)
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Location of config file",
			Value:   getDefaultConfigPath(),
		},

		&cli.BoolFlag{
			Name:  "ignore_dir_errors",
			Aliases: []string{"i"},
			Value: true,
			Usage: "Don't halt on errors while finding dirs",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "show debug output instead of UI",
		},
	}
	app.Action = func(c *cli.Context) error {

		config, err := scanner.ParseConfigFile(c.String("config"), defaultConfig)
		if c.Args().Len() > 0 {

			fmt.Println("Arguments given, skipping config")
			config.ScanDirs.Include = c.Args().Slice()

		} else {
			if err != nil {
				return err
			}
			for i := range config.ScanDirs.Include {
				config.ScanDirs.Include[i] = os.ExpandEnv(config.ScanDirs.Include[i])
			}
			for i := range config.ScanDirs.Exclude {
				config.ScanDirs.Exclude[i] = os.ExpandEnv(config.ScanDirs.Exclude[i])
			}
		}

		if c.Bool("debug") {
			var projects scanner.ProjectMap
			projects, err = scanner.Scan(config, c.Bool("ignore_dir_errors"))
			if err != nil {
				panic(err)
			}

			for r, st := range projects {
				fmt.Printf("%-40s %v\n", r, st.ScanTime)
			}
			return nil
		}

		err = ui.Run(config, c.Bool("ignore_dir_errors"), version)
		if err != nil {
			return err
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}
