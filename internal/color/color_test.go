package color

import "testing"

func TestParse(t *testing.T) {
	if c, err := Parse("red"); err != nil || !c.Set() || c.idx != 1 {
		t.Errorf("Parse(red) = %v, %v", c, err)
	}
	if c, err := Parse("196"); err != nil || c.idx != 196 {
		t.Errorf("Parse(196) = %v, %v", c, err)
	}
	if c, err := Parse(""); err != nil || c.Set() {
		t.Errorf("Parse(empty) = %v, %v; want unset, no error", c, err)
	}
	if _, err := Parse("notacolor"); err == nil {
		t.Error("Parse(notacolor) should error")
	}
	if _, err := Parse("300"); err == nil {
		t.Error("Parse(300) out of range should error")
	}
}

func TestParseHex(t *testing.T) {
	c, err := Parse("#ff8800")
	if err != nil || !c.set || !c.rgb || c.r != 0xff || c.g != 0x88 || c.b != 0x00 {
		t.Fatalf("Parse(#ff8800) = %+v, err %v", c, err)
	}

	if got, want := FG(c).Apply(On, "x"), "\x1b[38;2;255;136;0mx\x1b[0m"; got != want {
		t.Errorf("hex Apply = %q, want %q", got, want)
	}

	if short, _ := Parse("#f80"); short != c {
		t.Errorf("Parse(#f80) = %+v, want same as #ff8800 %+v", short, c)
	}
	for _, bad := range []string{"#ff", "#gggggg", "#12345"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) should error", bad)
		}
	}
}

func TestApply(t *testing.T) {
	s := FG(MustParse("green"))
	if got := s.Apply(Off, "x"); got != "x" {
		t.Errorf("Apply(Off) = %q, want unchanged", got)
	}
	if got, want := s.Apply(On, "x"), "\x1b[38;5;2mx\x1b[0m"; got != want {
		t.Errorf("Apply(On) = %q, want %q", got, want)
	}

	if got := (Style{}).Apply(On, "x"); got != "x" {
		t.Errorf("Apply(On, empty style) = %q, want unchanged", got)
	}
}

func TestThresholdPick(t *testing.T) {
	th := DefaultThreshold()
	cases := []struct {
		pct  float64
		want Color
	}{
		{0, th.OK}, {50, th.OK}, {50.1, th.WarnC}, {70, th.WarnC}, {70.1, th.DangerC}, {100, th.DangerC},
	}
	for _, c := range cases {
		if got := th.Pick(c.pct); got != c.want {
			t.Errorf("Pick(%v) = %+v, want %+v", c.pct, got, c.want)
		}
	}
}
