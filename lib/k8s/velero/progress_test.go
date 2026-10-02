package velero

import "testing"

// Installing Velero first takes the front of the progress bar, so the migration
// flow's own 0-100 reporting has to be squeezed into what is left. Without the
// mapping the bar jumps backwards from 20 to 5 when the migration starts.
func TestProgressScale(t *testing.T) {
	cases := []struct {
		name     string
		scale    progressScale
		progress int
		want     int
	}{
		{"zero value passes through", progressScale{}, 5, 5},
		{"zero value passes through at the end", progressScale{}, 100, 100},
		{"install reserved: start", progressScale{base: 20, span: 80}, 5, 24},
		{"install reserved: middle", progressScale{base: 20, span: 80}, 50, 60},
		{"install reserved: end", progressScale{base: 20, span: 80}, 100, 100},
		{"never below the base", progressScale{base: 20, span: 80}, 0, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.scale.at(tc.progress); got != tc.want {
				t.Errorf("progressScale%+v.at(%d) = %d, want %d", tc.scale, tc.progress, got, tc.want)
			}
		})
	}
}

// The migration flow reports in increasing order, so the mapped values must
// stay in order too whatever span it was given.
func TestProgressScale_Monotonic(t *testing.T) {
	scale := progressScale{base: 20, span: 80}

	previous := -1
	for _, progress := range []int{5, 15, 20, 30, 40, 65, 70, 72, 75, 80, 85, 90, 100} {
		got := scale.at(progress)
		if got < previous {
			t.Fatalf("at(%d) = %d went backwards from %d", progress, got, previous)
		}
		if got > 100 {
			t.Fatalf("at(%d) = %d exceeded 100", progress, got)
		}
		previous = got
	}
}
