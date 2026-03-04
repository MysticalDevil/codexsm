package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/MysticalDevil/codexsm/config"
	"github.com/spf13/cobra"
)

type doctorLevel string

const (
	doctorPass doctorLevel = "pass"
	doctorWarn doctorLevel = "warn"
	doctorFail doctorLevel = "fail"
)

type doctorCheck struct {
	Name   string
	Level  doctorLevel
	Detail string
}

func newDoctorCmd() *cobra.Command {
	var strict bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run local environment and configuration checks",
		Long: "Run local checks for codexsm runtime prerequisites.\n\n" +
			"This command validates config and storage paths.",
		Example: "  codexsm doctor\n" +
			"  codexsm doctor --strict",
		RunE: func(cmd *cobra.Command, args []string) error {
			checks := runDoctorChecks()
			out := renderDoctorChecks(checks, shouldUseColor("auto", cmd.OutOrStdout()))
			if _, err := fmt.Fprint(cmd.OutOrStdout(), out); err != nil {
				return err
			}
			if strict {
				for _, c := range checks {
					if c.Level == doctorFail || c.Level == doctorWarn {
						return WithExitCode(fmt.Errorf("doctor check failed: %s (%s)", c.Name, c.Level), 1)
					}
				}
			}
			for _, c := range checks {
				if c.Level == doctorFail {
					return WithExitCode(fmt.Errorf("doctor check failed: %s", c.Name), 1)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "treat warnings as failures")
	return cmd
}

func runDoctorChecks() []doctorCheck {
	checks := make([]doctorCheck, 0, 8)

	checks = append(checks, checkConfigFile())

	sessionsRoot, sessionsErr := runtimeSessionsRoot()
	checks = append(checks, checkDir("sessions_root", sessionsRoot, sessionsErr))

	trashRoot, trashErr := runtimeTrashRoot()
	checks = append(checks, checkDir("trash_root", trashRoot, trashErr))

	logFile, logErr := runtimeLogFile()
	checks = append(checks, checkLogFile(logFile, logErr))
	return checks
}

func checkConfigFile() doctorCheck {
	p, err := config.AppConfigPath()
	if err != nil {
		return doctorCheck{Name: "config", Level: doctorFail, Detail: err.Error()}
	}
	_, err = os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: "config", Level: doctorWarn, Detail: fmt.Sprintf("missing (optional): %s", p)}
		}
		return doctorCheck{Name: "config", Level: doctorFail, Detail: err.Error()}
	}
	cfg, err := config.LoadAppConfig()
	if err != nil {
		return doctorCheck{Name: "config", Level: doctorFail, Detail: err.Error()}
	}
	if strings.TrimSpace(cfg.SessionsRoot) == "" && strings.TrimSpace(cfg.TrashRoot) == "" && strings.TrimSpace(cfg.LogFile) == "" {
		return doctorCheck{Name: "config", Level: doctorPass, Detail: "loaded (no overrides)"}
	}
	return doctorCheck{Name: "config", Level: doctorPass, Detail: "loaded"}
}

func checkDir(name, path string, pathErr error) doctorCheck {
	if pathErr != nil {
		return doctorCheck{Name: name, Level: doctorFail, Detail: pathErr.Error()}
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: name, Level: doctorWarn, Detail: fmt.Sprintf("missing: %s", path)}
		}
		return doctorCheck{Name: name, Level: doctorFail, Detail: err.Error()}
	}
	if !info.IsDir() {
		return doctorCheck{Name: name, Level: doctorFail, Detail: fmt.Sprintf("not a directory: %s", path)}
	}
	if writable, msg := isWritableDir(path); !writable {
		return doctorCheck{Name: name, Level: doctorWarn, Detail: msg}
	}
	return doctorCheck{Name: name, Level: doctorPass, Detail: path}
}

func checkLogFile(path string, pathErr error) doctorCheck {
	if pathErr != nil {
		return doctorCheck{Name: "log_file", Level: doctorFail, Detail: pathErr.Error()}
	}
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return doctorCheck{Name: "log_file", Level: doctorWarn, Detail: fmt.Sprintf("parent dir missing: %s", dir)}
		}
		return doctorCheck{Name: "log_file", Level: doctorFail, Detail: err.Error()}
	}
	if !info.IsDir() {
		return doctorCheck{Name: "log_file", Level: doctorFail, Detail: fmt.Sprintf("parent is not directory: %s", dir)}
	}
	if writable, msg := isWritableDir(dir); !writable {
		return doctorCheck{Name: "log_file", Level: doctorWarn, Detail: msg}
	}
	return doctorCheck{Name: "log_file", Level: doctorPass, Detail: path}
}

func isWritableDir(path string) (bool, string) {
	f, err := os.CreateTemp(path, ".codexsm-doctor-*")
	if err != nil {
		return false, fmt.Sprintf("not writable: %s (%v)", path, err)
	}
	name := f.Name()
	if closeErr := f.Close(); closeErr != nil {
		return false, fmt.Sprintf("close temp file failed: %v", closeErr)
	}
	if rmErr := os.Remove(name); rmErr != nil {
		return false, fmt.Sprintf("cleanup temp file failed: %v", rmErr)
	}
	return true, path
}

func renderDoctorChecks(checks []doctorCheck, color bool) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "CHECK\tSTATUS\tDETAIL")
	for _, c := range checks {
		status := string(c.Level)
		if color {
			switch c.Level {
			case doctorPass:
				status = colorize(status, ansiGreen, true)
			case doctorWarn:
				status = colorize(status, ansiYellow, true)
			case doctorFail:
				status = colorize(status, ansiRed, true)
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", c.Name, status, c.Detail)
	}
	_ = w.Flush()
	return buf.String()
}
