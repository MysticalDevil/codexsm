package cli

import (
	"time"

	"github.com/MysticalDevil/codexsm/audit"
	"github.com/MysticalDevil/codexsm/usecase"
)

type clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

type defaultAuditSink struct{}

func (defaultAuditSink) NewBatchID() (string, error) {
	return audit.NewBatchID()
}

func (defaultAuditSink) WriteActionLog(logFile string, rec audit.ActionRecord) error {
	return audit.WriteActionLog(logFile, rec)
}

var runtimeClock clock = systemClock{}
var runtimeAuditSink usecase.AuditSink = defaultAuditSink{}
