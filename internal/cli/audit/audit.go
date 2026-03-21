package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leszko11/google-play-console-cli/internal/cli/shared"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const defaultAuditFile = ".gpc/audit.log"

type Deps struct {
	Stdout io.Writer
	Stderr io.Writer
	// HomeDir overrides the home directory for testing
	HomeDir string
	// Clock allows tests to inject a fake time
	Clock func() time.Time
}

type auditEntry struct {
	Timestamp   string `json:"timestamp"`
	Command     string `json:"command"`
	PackageName string `json:"packageName,omitempty"`
	User        string `json:"user,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

func NewCommand(deps Deps) *ffcli.Command {
	deps = withDefaults(deps)

	return &ffcli.Command{
		Name:      "audit",
		ShortHelp: "View and manage the local CLI audit trail",
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			newLogCommand(deps),
			newShowCommand(deps),
		},
	}
}

func newLogCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var command, packageName, user, detail string
	fs.StringVar(&command, "command", "", "Command that was executed")
	fs.StringVar(&packageName, "package-name", "", "Package name (optional)")
	fs.StringVar(&user, "user", "", "User who ran the command (optional)")
	fs.StringVar(&detail, "detail", "", "Additional detail (optional)")

	return &ffcli.Command{
		Name:      "log",
		ShortHelp: "Append an entry to the audit trail",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(_ context.Context, _ []string) error {
			if strings.TrimSpace(command) == "" {
				return shared.UsageErrorf("--command is required")
			}

			entry := auditEntry{
				Timestamp:   deps.Clock().UTC().Format(time.RFC3339),
				Command:     command,
				PackageName: packageName,
				User:        user,
				Detail:      detail,
			}

			path := auditFilePath(deps.HomeDir)
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0700); err != nil {
				return fmt.Errorf("create audit directory: %w", err)
			}

			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("open audit log: %w", err)
			}
			defer f.Close()

			b, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			if _, err := f.Write(append(b, '\n')); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(deps.Stdout, "logged: %s\n", command); err != nil {
				return err
			}
			return nil
		},
	}
}

func newShowCommand(deps Deps) *ffcli.Command {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(deps.Stderr)

	var output string
	var last int
	fs.StringVar(&output, "output", "", "Output format: json, table, markdown")
	fs.IntVar(&last, "last", 0, "Show only the last N entries")

	return &ffcli.Command{
		Name:      "show",
		ShortHelp: "Display the audit trail",
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(_ context.Context, _ []string) error {
			resolvedOutput := shared.ResolveOutput(output)
			if resolvedOutput != "json" && resolvedOutput != "table" && resolvedOutput != "markdown" {
				return shared.UsageErrorf("unsupported output format %q", resolvedOutput)
			}

			path := auditFilePath(deps.HomeDir)
			f, err := os.Open(path)
			if err != nil {
				if os.IsNotExist(err) {
					return writeEntries(deps.Stdout, resolvedOutput, []auditEntry{})
				}
				return fmt.Errorf("open audit log: %w", err)
			}
			defer f.Close()

			var entries []auditEntry
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				var entry auditEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					continue // skip malformed lines
				}
				entries = append(entries, entry)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read audit log: %w", err)
			}

			if last > 0 && last < len(entries) {
				entries = entries[len(entries)-last:]
			}

			return writeEntries(deps.Stdout, resolvedOutput, entries)
		},
	}
}

func writeEntries(out io.Writer, format string, entries []auditEntry) error {
	switch format {
	case "json":
		return shared.WriteJSON(out, entries)
	case "table":
		return writeShowTable(out, entries)
	case "markdown":
		return writeShowMarkdown(out, entries)
	default:
		return shared.UsageErrorf("unsupported output format %q", format)
	}
}

func writeShowTable(out io.Writer, entries []auditEntry) error {
	if _, err := fmt.Fprintln(out, "TIMESTAMP\tCOMMAND\tPACKAGE\tUSER\tDETAIL"); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", e.Timestamp, e.Command, e.PackageName, e.User, e.Detail); err != nil {
			return err
		}
	}
	return nil
}

func writeShowMarkdown(out io.Writer, entries []auditEntry) error {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{e.Timestamp, e.Command, e.PackageName, e.User, e.Detail})
	}
	return shared.WriteMarkdownTable(out, []string{"timestamp", "command", "package", "user", "detail"}, rows)
}

func auditFilePath(homeDir string) string {
	if homeDir == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		} else {
			homeDir = h
		}
	}
	return filepath.Join(homeDir, defaultAuditFile)
}

func withDefaults(deps Deps) Deps {
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}
	if deps.Stderr == nil {
		deps.Stderr = os.Stderr
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	return deps
}
