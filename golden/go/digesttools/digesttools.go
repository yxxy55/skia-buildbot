// digesttools are utility functions for answering questions about digests.
package digesttools

import (
	"math"

	"github.com/skia-dev/glog"
	"go.skia.org/infra/golden/go/diff"
	"go.skia.org/infra/golden/go/expstorage"
	"go.skia.org/infra/golden/go/types"
)

// Closest describes one digest that is the closest another digest.
type Closest struct {
	Digest     string  `json:"digest"`     // The closest digest, empty if there are no digests to compare to.
	Diff       float32 `json:"diff"`       // A percent value.
	DiffPixels float32 `json:"diffPixels"` // A percent value.
	MaxRGBA    []int   `json:"maxRGBA"`
}

func newClosest() *Closest {
	return &Closest{
		Diff:       math.MaxFloat32,
		DiffPixels: math.MaxFloat32,
		MaxRGBA:    []int{},
	}
}

// ClosestDigest returns the closest digest of type 'label' to 'digest', or "" if there aren't any positive digests.
//
// If no digest of type 'label' is found then Closest.Digest is the empty string.
func ClosestDigest(test string, digest string, exp *expstorage.Expectations, diffStore diff.DiffStore, label types.Label) *Closest {
	ret := newClosest()
	selected := []string{}
	if e, ok := exp.Tests[test]; ok {
		for d, l := range e {
			if l == label {
				selected = append(selected, d)
			}
		}
	}
	if diffMetrics, err := diffStore.Get(digest, selected); err != nil {
		glog.Errorf("ClosestDigest: Failed to get diff: %s", err)
		return ret
	} else {
		for digest, diff := range diffMetrics {
			if delta := combinedDiffMetric(diff.PixelDiffPercent, diff.MaxRGBADiffs); delta < ret.Diff {
				ret.Digest = digest
				ret.Diff = delta
				ret.DiffPixels = diff.PixelDiffPercent
				ret.MaxRGBA = diff.MaxRGBADiffs
			}
		}
		return ret
	}
}

// combinedDiffMetric returns a value in [0, 1] that represents how large
// the diff is between two images.
func combinedDiffMetric(pixelDiffPercent float32, maxRGBA []int) float32 {
	// Turn maxRGBA into a percent by taking the root mean square difference from
	// [0, 0, 0, 0].
	sum := 0.0
	for _, c := range maxRGBA {
		sum += float64(c) * float64(c)
	}
	normalizedRGBA := math.Sqrt(sum/float64(len(maxRGBA))) / 255.0
	// We take the sqrt of (pixelDiffPercent * normalizedRGBA) to straigten out
	// the curve, i.e. think about what a plot of x^2 would look like in the
	// range [0, 1].
	return float32(math.Sqrt(float64(pixelDiffPercent) * normalizedRGBA))
}
