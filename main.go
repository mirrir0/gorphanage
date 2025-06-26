package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information (set by build)
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	// CLI flags
	outputsJSON  bool
	verbose      bool
	configFile   string
	exclude      []string
	includeTests bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "gorphanage [flags] <project-path>",
	Short: "🏠 A home for finding Go's lost code",
	Long: `Gorphanage finds orphaned code in Go projects using advanced reachability analysis.

It traces execution paths from main() functions to identify symbols that are
genuinely unreachable, helping you clean up dead code with confidence.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Example: `  # Analyze current directory
  gorphanage .

  # Output JSON for tooling
  gorphanage --json ./cmd/myapp

  # Exclude specific packages
  gorphanage --exclude vendor,generated .

  # Include test files in analysis
  gorphanage --include-tests .

  # Verbose output with detailed progress
  gorphanage --verbose .`,
	Args: cobra.ExactArgs(1),
	RunE: runAnalysis,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.gorphanage.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Analysis flags
	rootCmd.Flags().BoolVar(&outputsJSON, "json", false, "output results in JSON format")
	rootCmd.Flags().StringSliceVarP(&exclude, "exclude", "e", []string{}, "exclude packages matching these patterns")
	rootCmd.Flags().BoolVar(&includeTests, "include-tests", false, "include test files in analysis")

	// Bind flags to viper
	viper.BindPFlag("json", rootCmd.Flags().Lookup("json"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("exclude", rootCmd.Flags().Lookup("exclude"))
	viper.BindPFlag("include-tests", rootCmd.Flags().Lookup("include-tests"))

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gorphanage" (without extension)
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gorphanage")
	}

	// Environment variable support
	viper.SetEnvPrefix("GORPHANAGE")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	}
}

func runAnalysis(cmd *cobra.Command, args []string) error {
	projectPath := args[0]

	// Resolve absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	// Create config from flags and viper settings
	config := &Config{
		ProjectPath:  absPath,
		OutputJSON:   viper.GetBool("json"),
		Verbose:      viper.GetBool("verbose"),
		Exclude:      viper.GetStringSlice("exclude"),
		IncludeTests: viper.GetBool("include-tests"),
	}

	if config.Verbose && !config.OutputJSON {
		fmt.Printf("🔍 Analyzing project at: %s\n", config.ProjectPath)
		if len(config.Exclude) > 0 {
			fmt.Printf("📋 Excluding patterns: %v\n", config.Exclude)
		}
		if config.IncludeTests {
			fmt.Printf("🧪 Including test files in analysis\n")
		}
	}

	// Create and run analyzer
	analyzer := NewAnalyzer(config)
	result, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Output results
	if config.OutputJSON {
		return outputJSON(result)
	}

	analyzer.PrintResults(result)
	return nil
}

func outputJSON(result *AnalysisResult) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print detailed version information including build metadata",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Gorphanage %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		fmt.Printf("Go version: %s\n", getGoVersion())
	},
}

// Config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  "Manage Gorphanage configuration settings",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display the current configuration values from all sources",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current configuration:")
		fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
		fmt.Printf("JSON output: %v\n", viper.GetBool("json"))
		fmt.Printf("Verbose: %v\n", viper.GetBool("verbose"))
		fmt.Printf("Exclude patterns: %v\n", viper.GetStringSlice("exclude"))
		fmt.Printf("Include tests: %v\n", viper.GetBool("include-tests"))
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  "Create a default configuration file in the home directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configPath := filepath.Join(home, ".gorphanage.yaml")

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config file already exists at %s", configPath)
		}

		// Default configuration
		defaultConfig := `# Gorphanage configuration file
# See https://github.com/yourusername/gorphanage for documentation

# Output format
json: false
verbose: false

# Analysis options
include-tests: false

# Exclude patterns (glob patterns for package paths)
exclude:
  - "vendor/*"
  - "*.pb.go"
  - "*_generated.go"

# Advanced options
# max-depth: 10
# timeout: "5m"
`

		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("✅ Created config file at %s\n", configPath)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
}

func getGoVersion() string {
	// This would be set during build, or detected at runtime
	return "go version not available"
}
