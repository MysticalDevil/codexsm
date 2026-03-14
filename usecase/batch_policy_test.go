package usecase

import "testing"

func TestEffectiveMaxBatch(t *testing.T) {
	if got := EffectiveMaxBatch(false, 777, true); got != DefaultMaxBatchDryRun {
		t.Fatalf("unexpected dry-run default max-batch: %d", got)
	}

	if got := EffectiveMaxBatch(false, 777, false); got != DefaultMaxBatchReal {
		t.Fatalf("unexpected real default max-batch: %d", got)
	}

	if got := EffectiveMaxBatch(true, 123, true); got != 123 {
		t.Fatalf("expected configured max-batch override, got %d", got)
	}
}

func TestEffectiveMaxBatchWithDefaults(t *testing.T) {
	if got := EffectiveMaxBatchWithDefaults(false, 999, false, 100, 1000); got != 100 {
		t.Fatalf("expected real default, got %d", got)
	}

	if got := EffectiveMaxBatchWithDefaults(false, 999, true, 100, 1000); got != 1000 {
		t.Fatalf("expected dry-run default, got %d", got)
	}

	if got := EffectiveMaxBatchWithDefaults(true, 321, true, 100, 1000); got != 321 {
		t.Fatalf("expected explicit max-batch, got %d", got)
	}
}
