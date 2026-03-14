package usecase

const (
	DefaultMaxBatchReal   = 50
	DefaultMaxBatchDryRun = 500
)

func EffectiveMaxBatch(flagChanged bool, configured int, dryRun bool) int {
	return EffectiveMaxBatchWithDefaults(flagChanged, configured, dryRun, DefaultMaxBatchReal, DefaultMaxBatchDryRun)
}

func EffectiveMaxBatchWithDefaults(flagChanged bool, configured int, dryRun bool, realDefault, dryRunDefault int) int {
	if flagChanged {
		return configured
	}

	if dryRun {
		return dryRunDefault
	}

	return realDefault
}
