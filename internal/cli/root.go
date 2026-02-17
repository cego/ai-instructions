package cli

import (
	"os"
	"strings"

	"github.com/cego/ai-instructions/internal/config"
	"github.com/cego/ai-instructions/internal/exitcodes"
	"github.com/cego/ai-instructions/internal/registry"
	"github.com/cego/ai-instructions/internal/ui"
	"github.com/spf13/cobra"
)

// App is the dependency container for all CLI commands.
type App struct {
	rootCmd  *cobra.Command
	version  string
	commit   string
	date     string
	config   *config.Config
	output   *ui.Output
	projectDir  string
	registryURL string
	branch      string
	token       string
	debug       bool
}

// NewApp creates the root command and registers all subcommands.
func NewApp(version, commit, date string) *App {
	app := &App{
		version: version,
		commit:  commit,
		date:    date,
		output:  ui.NewOutput(),
	}

	root := &cobra.Command{
		Use:   "ai-instructions",
		Short: "Package manager for AI coding instruction files",
		Long:  "Manages company-wide AI coding instruction files (.md) across project repositories.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if envURL := os.Getenv("AI_INSTRUCTIONS_REGISTRY"); envURL != "" && app.registryURL == "" {
				app.registryURL = envURL
			}
			if envBranch := os.Getenv("AI_INSTRUCTIONS_BRANCH"); envBranch != "" && app.branch == "" {
				app.branch = envBranch
			}
			if envToken := os.Getenv("AI_INSTRUCTIONS_TOKEN"); envToken != "" && app.token == "" {
				app.token = envToken
			}
			if os.Getenv("AI_INSTRUCTIONS_DEBUG") != "" {
				app.debug = true
			}
			if os.Getenv("AI_INSTRUCTIONS_NO_COLOR") != "" || os.Getenv("NO_COLOR") != "" {
				app.output.SetNoColor(true)
			}

			// Eagerly load config (ignore errors — commands that need it will call RequireProject)
			_ = app.LoadProjectConfig()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&app.registryURL, "registry", "", "registry URL (overrides AI_INSTRUCTIONS_REGISTRY)")
	root.PersistentFlags().StringVar(&app.branch, "branch", "", "registry branch (default: master, overrides AI_INSTRUCTIONS_BRANCH)")
	root.PersistentFlags().StringVar(&app.token, "token", "", "auth token (overrides AI_INSTRUCTIONS_TOKEN)")
	root.PersistentFlags().BoolVar(&app.debug, "debug", false, "enable debug logging")
	root.PersistentFlags().StringVar(&app.projectDir, "dir", ".", "project directory")

	root.AddCommand(
		app.newInitCmd(),
		app.newSyncCmd(),
		app.newVerifyCmd(),
		app.newListCmd(),
		app.newVersionCmd(),
	)

	app.rootCmd = root
	return app
}

// Execute runs the root command.
func (a *App) Execute() error {
	return a.rootCmd.Execute()
}

// LoadProjectConfig loads the config file. Falls back to migration from old settings.
// If a separate old lockfile exists and the config has no resolved data, absorbs it.
// Returns nil error if no config is found.
func (a *App) LoadProjectConfig() error {
	if config.ConfigExists(a.projectDir) {
		c, err := config.LoadConfig(a.projectDir)
		if err != nil {
			return err
		}
		a.config = c

		// Absorb old lockfile if config has no resolved data yet
		if a.config.Resolved == nil && config.OldLockfileExists(a.projectDir) {
			if err := config.AbsorbLockfile(a.projectDir, a.config); err != nil {
				return err
			}
		}

		return nil
	}

	// Try migrating from old settings
	if config.OldSettingsExists(a.projectDir) {
		c, err := config.MigrateFromOldSettings(a.projectDir)
		if err != nil {
			return err
		}
		a.config = c
		return nil
	}

	return nil
}

// RequireProject loads config and returns an error if it doesn't exist or has no resolved data.
func (a *App) RequireProject() error {
	if a.config == nil {
		if err := a.LoadProjectConfig(); err != nil {
			return err
		}
	}
	if a.config == nil {
		return &ExitError{
			Code:    exitcodes.ConfigError,
			Message: "no " + config.ConfigFile + " found — run 'ai-instructions init' first",
		}
	}

	if a.config.Resolved == nil {
		return &ExitError{
			Code:    exitcodes.ConfigError,
			Message: "no resolved stacks found — run 'ai-instructions sync' first",
		}
	}

	return nil
}

// getBranch returns the effective branch name.
func (a *App) getBranch() string {
	if a.branch != "" {
		return a.branch
	}
	if a.config != nil && a.config.Registry.Branch != "" {
		return a.config.Registry.Branch
	}
	return config.DefaultBranch
}

// getProjectURL returns the effective GitLab project URL (without branch path).
func (a *App) getProjectURL() string {
	base := a.registryURL
	if base == "" && a.config != nil {
		base = a.config.Registry.URL
	}
	if base == "" {
		base = config.DefaultRegistryURL
	}

	return strings.TrimRight(base, "/")
}

// getInstructionsDir returns the effective top-level instructions directory.
func (a *App) getInstructionsDir() string {
	if a.config != nil && a.config.InstructionsDir != "" {
		return a.config.InstructionsDir
	}
	return config.DefaultInstructionsDir
}

// getManagedDir returns the managed subdirectory path within the instructions dir.
// This is where registry-downloaded files live and can be safely wiped on sync.
func (a *App) getManagedDir() string {
	return a.getInstructionsDir() + "/" + config.ManagedDir
}

// newRegistryClient creates a registry client with the current settings.
func (a *App) newRegistryClient() (*registry.Client, error) {
	projectURL := a.getProjectURL()
	if projectURL == "" {
		return nil, &ExitError{
			Code:    exitcodes.ConfigError,
			Message: "registry URL not set — use --registry flag or AI_INSTRUCTIONS_REGISTRY env var",
		}
	}
	opts := []registry.Option{
		registry.WithProjectURL(projectURL),
		registry.WithBranch(a.getBranch()),
	}
	if a.token != "" {
		opts = append(opts, registry.WithToken(a.token))
	}
	return registry.NewClient(opts...), nil
}

func (a *App) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			a.output.Info("ai-instructions %s (commit: %s, built: %s)", a.version, a.commit, a.date)
		},
	}
}

// ExitError represents an error with a specific exit code.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// debugf prints a debug message if debug mode is enabled.
func (a *App) debugf(format string, args ...interface{}) {
	if a.debug {
		a.output.Debug(format, args...)
	}
}
