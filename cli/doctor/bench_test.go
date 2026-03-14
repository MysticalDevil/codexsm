package doctor_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MysticalDevil/codexsm/cli"
	cliutil "github.com/MysticalDevil/codexsm/cli/util"
)

func BenchmarkDoctorRiskJSON(b *testing.B) {
	root := prepareDoctorRiskBenchRoot(b, 800)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd := cli.NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"doctor", "risk", "--sessions-root", root, "--format", "json", "--sample-limit", "5", "--integrity-check=false"})

		err := cmd.Execute()
		if err == nil {
			b.Fatal("expected exit code 1 when benchmark dataset contains risk")
		}

		var ex *cliutil.ExitError
		if !errors.As(err, &ex) || ex.ExitCode() != 1 {
			b.Fatalf("unexpected error: %v", err)
		}

		if stdout.Len() == 0 || !strings.Contains(stdout.String(), `"risk_total":`) {
			b.Fatalf("unexpected doctor risk output: %q", stdout.String())
		}
	}
}

func prepareDoctorRiskBenchRoot(b *testing.B, count int) string {
	b.Helper()
	root := b.TempDir()

	base := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		created := base.Add(time.Duration(i) * time.Minute)

		dayDir := filepath.Join(root, created.Format("2006"), created.Format("01"), created.Format("02"))
		if err := os.MkdirAll(dayDir, 0o755); err != nil {
			b.Fatalf("mkdir bench day dir: %v", err)
		}

		path := filepath.Join(dayDir, fmt.Sprintf("doctor-bench-%04d.jsonl", i))

		content := doctorRiskBenchContent(created, i)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			b.Fatalf("write doctor bench session: %v", err)
		}
	}

	return root
}

func doctorRiskBenchContent(created time.Time, i int) string {
	switch {
	case i%97 == 0:
		return `{"type":"session_meta","payload":{"id":"broken"` + "\n" + `not-json-line` + "\n"
	case i%41 == 0:
		return fmt.Sprintf(`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"missing meta %d"}]}}`+"\n", i)
	default:
		return fmt.Sprintf(
			`{"type":"session_meta","payload":{"id":"bench-%04d","timestamp":"%s","cwd":"/workspace/doctor/%02d"}}`+"\n"+
				`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"doctor risk benchmark message %04d"}]}}`+"\n",
			i,
			created.Format(time.RFC3339Nano),
			i%12,
			i,
		)
	}
}
