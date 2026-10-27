package record

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Root returns a cobra root command with --log-level and a version subcommand.
func Root(name, short string) *cobra.Command {
	var level string
	cmd := &cobra.Command{
		Use:          name,
		Short:        short,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return SetupLog(level)
		},
	}
	cmd.PersistentFlags().StringVar(&level, "log-level", "warn", "log level: debug|info|warn|error (stderr)")
	cmd.AddCommand(VersionCommand(name))
	return cmd
}

// VersionCommand prints the binary name, version, commit and build date.
func VersionCommand(binary string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version, commit and build date",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			v, commit, date := buildInfo()
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (commit %s, built %s)\n", binary, v, commit, date)
			return nil
		},
	}
}

// AddFromFlags registers --from-<name> FILE flags on cmd and binds them to froms.
func AddFromFlags(cmd *cobra.Command, froms []*From) {
	for _, f := range froms {
		name := "from-" + f.Name
		cmd.Flags().StringVar(&f.File, name, "", "read "+f.Name+" records from FILE (`-` for stdin)")
	}
}

// SetupLog configures slog text output on stderr. level is debug, info, warn or error.
func SetupLog(level string) error {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return fmt.Errorf("invalid --log-level %q (want debug|info|warn|error)", level)
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
	return nil
}

// LogStart logs the command, its input files and flags at debug level.
func LogStart(command string, files map[string]string, flags map[string]string) {
	attrs := []any{slog.String("command", command)}
	for k, v := range files {
		if v != "" {
			attrs = append(attrs, slog.String("file."+k, v))
		}
	}
	for k, v := range flags {
		attrs = append(attrs, slog.String("flag."+k, v))
	}
	slog.Debug("command start", attrs...)
}

func buildInfo() (version, commit, date string) {
	version, commit, date = "dev", "unknown", "unknown"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
			if len(commit) > 7 {
				commit = commit[:7]
			}
		case "vcs.time":
			date = s.Value
		}
	}
	return
}
