package layout

import "testing"

func TestNormalizeSize(t *testing.T) {
	w, h := NormalizeSize(0, 0)
	if w != 120 || h != 32 {
		t.Fatalf("unexpected default size: %dx%d", w, h)
	}

	w, h = NormalizeSize(140, 40)
	if w != 140 || h != 40 {
		t.Fatalf("unexpected passthrough size: %dx%d", w, h)
	}
}

func TestRenderWidth(t *testing.T) {
	if got := RenderWidth(0); got != 0 {
		t.Fatalf("RenderWidth(0)=%d, want 0", got)
	}
	if got := RenderWidth(1); got != 1 {
		t.Fatalf("RenderWidth(1)=%d, want 1", got)
	}
	if got := RenderWidth(140); got != 139 {
		t.Fatalf("RenderWidth(140)=%d, want 139", got)
	}
}

func TestIsTooSmall(t *testing.T) {
	if IsTooSmall(118, 30) != true {
		t.Fatal("width below minimum should be too small")
	}
	if IsTooSmall(120, 20) != true {
		t.Fatal("height below minimum should be too small")
	}
	if IsTooSmall(119, 30) != false {
		t.Fatal("expected enough terminal size")
	}
	// Unknown runtime size should not trigger warning path.
	if IsTooSmall(0, 0) != false {
		t.Fatal("zero size should not be treated as too small")
	}
}

func TestCompute(t *testing.T) {
	m := Compute(120, 32)
	if m.TotalW != 119 || m.TotalH != 32 {
		t.Fatalf("unexpected normalized size: %+v", m)
	}
	if m.KeysOuterH != 3 {
		t.Fatalf("unexpected keys height: %+v", m)
	}
	if m.LeftOuterW+m.GapW+m.RightOuterW > m.TotalW {
		t.Fatalf("horizontal overflow: %+v", m)
	}
	if m.InfoOuterH+m.PreviewOuterH != m.MainAreaH {
		t.Fatalf("vertical mismatch: %+v", m)
	}
	if m.LeftOuterW < 28 || m.RightOuterW < 36 {
		t.Fatalf("min width guard failed: %+v", m)
	}
}

func TestComputeUsesTwentyEightPercentLeftPane(t *testing.T) {
	m := Compute(140, 32)
	if m.LeftOuterW != 38 {
		t.Fatalf("expected ~28%% left pane, got %+v", m)
	}
}

func TestComputeDropsGapAtNarrowWidths(t *testing.T) {
	m := Compute(128, 32)
	if m.GapW != 0 {
		t.Fatalf("expected narrow layout gap=0, got %+v", m)
	}
}
