package cli

import (
	"github.com/MysticalDevil/codexsm/audit"
)

var (
	runtimeNewBatchID     = audit.NewBatchID
	runtimeWriteActionLog = audit.WriteActionLog
)
