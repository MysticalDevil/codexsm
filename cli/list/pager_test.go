package list

import "testing"

func TestApplyPagerChoice(t *testing.T) {
	type tc struct {
		name     string
		page     int
		pages    int
		in       string
		wantPage int
		wantAct  PagerAction
	}

	cases := []tc{
		{name: "next j", page: 0, pages: 5, in: "j\n", wantPage: 1, wantAct: PagerActionContinue},
		{name: "next enter", page: 0, pages: 5, in: "\n", wantPage: 1, wantAct: PagerActionContinue},
		{name: "back k", page: 3, pages: 5, in: "k\n", wantPage: 2, wantAct: PagerActionContinue},
		{name: "first g", page: 3, pages: 5, in: "g\n", wantPage: 0, wantAct: PagerActionContinue},
		{name: "last G", page: 1, pages: 5, in: "G\n", wantPage: 4, wantAct: PagerActionContinue},
		{name: "all", page: 2, pages: 5, in: "a\n", wantPage: 2, wantAct: PagerActionAll},
		{name: "quit", page: 2, pages: 5, in: "q\n", wantPage: 2, wantAct: PagerActionQuit},
		{name: "unknown defaults to next", page: 2, pages: 5, in: "x\n", wantPage: 3, wantAct: PagerActionContinue},
		{name: "back clamps at zero", page: 0, pages: 5, in: "k\n", wantPage: 0, wantAct: PagerActionContinue},
		{name: "next clamps at last", page: 4, pages: 5, in: "j\n", wantPage: 4, wantAct: PagerActionContinue},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPage, gotAct := ApplyPagerChoice(c.page, c.pages, c.in)
			if gotPage != c.wantPage || gotAct != c.wantAct {
				t.Fatalf("applyPagerChoice(%d,%d,%q) got (%d,%d), want (%d,%d)",
					c.page, c.pages, c.in, gotPage, gotAct, c.wantPage, c.wantAct)
			}
		})
	}
}
