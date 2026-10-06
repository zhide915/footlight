package color

type Threshold struct {
	Warn, Danger float64
	OK, WarnC    Color
	DangerC      Color
}

func DefaultThreshold() Threshold {
	return Threshold{
		Warn:    50,
		Danger:  70,
		OK:      MustParse("#98C379"),
		WarnC:   MustParse("#E5C07B"),
		DangerC: MustParse("#E06C75"),
	}
}

// Pick uses strict > comparisons, so a value sitting exactly on a boundary
// stays in the lower (safer) band.
func (t Threshold) Pick(pct float64) Color {
	switch {
	case pct > t.Danger:
		return t.DangerC
	case pct > t.Warn:
		return t.WarnC
	default:
		return t.OK
	}
}
