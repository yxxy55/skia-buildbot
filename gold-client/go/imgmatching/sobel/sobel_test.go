package sobel

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.skia.org/infra/gold-client/go/imgmatching/fuzzy"
	"go.skia.org/infra/gold-client/go/mocks"
	"go.skia.org/infra/golden/go/image/text"
)

// matcherTestCase represents a test case for the sobel.Matcher's Match() method.
type matcherTestCase struct {
	name        string
	inputImage1 image.Image
	inputImage2 image.Image

	// Matcher parameters.
	edgeThreshold       int
	maxDifferentPixels  int
	pixelDeltaThreshold int

	// Expected images passed to the embedded fuzzy.Matcher.
	expectedFuzzyMatcherInputImage1 image.Image
	expectedFuzzyMatcherInputImage2 image.Image

	expectImagesToMatch bool // Expected matcher output.

	// Debug information about the last matched pair of images.
	expectedSobelOutput            image.Image
	expectedImage1WithEdgesRemoved image.Image
	expectedImage2WithEdgesRemoved image.Image
	expectedNumDifferentPixels     int
	expectedMaxPixelDelta          int
}

// makeMatcherTestCases returns a slice of test cases shared by the TestMatcher_Match_* tests
// below.
func makeMatcherTestCases() []matcherTestCase {
	return []matcherTestCase{
		{
			name:                            "edge threshold 0xFF",
			inputImage1:                     text.MustToNRGBA(image1),
			inputImage2:                     text.MustToNRGBA(image2),
			edgeThreshold:                   0xFF,
			maxDifferentPixels:              3,
			pixelDeltaThreshold:             10,
			expectedFuzzyMatcherInputImage1: text.MustToNRGBA(image1),
			expectedFuzzyMatcherInputImage2: text.MustToNRGBA(image2),
			expectImagesToMatch:             false, // 10 pixels off, max per-channel delta sum of 36.
			expectedSobelOutput:             text.MustToGray(image1Sobel),
			expectedImage1WithEdgesRemoved:  text.MustToNRGBA(image1),
			expectedImage2WithEdgesRemoved:  text.MustToNRGBA(image2),
			expectedNumDifferentPixels:      10,
			expectedMaxPixelDelta:           36,
		},
		{
			name:                            "edge threshold 0xAA",
			inputImage1:                     text.MustToNRGBA(image1),
			inputImage2:                     text.MustToNRGBA(image2),
			edgeThreshold:                   0xAA,
			maxDifferentPixels:              3,
			pixelDeltaThreshold:             10,
			expectedFuzzyMatcherInputImage1: text.MustToNRGBA(image1NoEdgesAbove0xAA),
			expectedFuzzyMatcherInputImage2: text.MustToNRGBA(image2NoEdgesAbove0xAA),
			expectImagesToMatch:             false, // 5 pixels off, max per-channel delta sum of 15.
			expectedSobelOutput:             text.MustToGray(image1Sobel),
			expectedImage1WithEdgesRemoved:  text.MustToNRGBA(image1NoEdgesAbove0xAA),
			expectedImage2WithEdgesRemoved:  text.MustToNRGBA(image2NoEdgesAbove0xAA),
			expectedNumDifferentPixels:      5,
			expectedMaxPixelDelta:           15,
		},
		{
			name:                            "edge threshold 0x66",
			inputImage1:                     text.MustToNRGBA(image1),
			inputImage2:                     text.MustToNRGBA(image2),
			edgeThreshold:                   0x66,
			maxDifferentPixels:              3,
			pixelDeltaThreshold:             10,
			expectedFuzzyMatcherInputImage1: text.MustToNRGBA(image1NoEdgesAbove0x66),
			expectedFuzzyMatcherInputImage2: text.MustToNRGBA(image2NoEdgesAbove0x66),
			expectImagesToMatch:             true, // 1 pixel off, max per-channel delta sum of 9.
			expectedSobelOutput:             text.MustToGray(image1Sobel),
			expectedImage1WithEdgesRemoved:  text.MustToNRGBA(image1NoEdgesAbove0x66),
			expectedImage2WithEdgesRemoved:  text.MustToNRGBA(image2NoEdgesAbove0x66),
			expectedNumDifferentPixels:      1,
			expectedMaxPixelDelta:           9,
		},
		{
			name:                            "edge threshold 0x00",
			inputImage1:                     text.MustToNRGBA(image1),
			inputImage2:                     text.MustToNRGBA(image2),
			edgeThreshold:                   0x00,
			maxDifferentPixels:              3,
			pixelDeltaThreshold:             10,
			expectedFuzzyMatcherInputImage1: text.MustToNRGBA(image1NoEdgesAbove0x00),
			expectedFuzzyMatcherInputImage2: text.MustToNRGBA(image2NoEdgesAbove0x00),
			expectImagesToMatch:             true, // The above images are identical.
			expectedSobelOutput:             text.MustToGray(image1Sobel),
			expectedImage1WithEdgesRemoved:  text.MustToNRGBA(image1NoEdgesAbove0x00),
			expectedImage2WithEdgesRemoved:  text.MustToNRGBA(image2NoEdgesAbove0x00),
			expectedNumDifferentPixels:      0,
			expectedMaxPixelDelta:           0,
		},
	}
}

// TestMatcher_Match_MockFuzzyMatcher_CallsFuzzyMatcherWithExpectedInputImages tests
// sobel.Matcher's Match() method in isolation with respect to the embedded fuzzy.Matcher.
func TestMatcher_Match_MockFuzzyMatcher_CallsFuzzyMatcherWithExpectedInputImages(t *testing.T) {

	for _, tc := range makeMatcherTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			fuzzyMatcher := &mocks.Matcher{}

			// Return value does not matter, we're only testing that the right inputs are passed.
			fuzzyMatcher.On("Match", tc.expectedFuzzyMatcherInputImage1, tc.expectedFuzzyMatcherInputImage2).Return(true)

			sobelMatcher := Matcher{
				EdgeThreshold:          tc.edgeThreshold,
				fuzzyMatcherForTesting: fuzzyMatcher,
			}

			assert.True(t, sobelMatcher.Match(tc.inputImage1, tc.inputImage2))
			fuzzyMatcher.AssertExpectations(t)
		})
	}
}

// TestMatcher_Match_Success tests sobel.Matcher's Match() method using a real fuzzy.Matcher.
func TestMatcher_Match_Success(t *testing.T) {

	for _, tc := range makeMatcherTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			matcher := Matcher{
				Matcher: fuzzy.Matcher{
					MaxDifferentPixels:  tc.maxDifferentPixels,
					PixelDeltaThreshold: tc.pixelDeltaThreshold,
				},
				EdgeThreshold: tc.edgeThreshold,
			}

			assert.Equal(t, tc.expectImagesToMatch, matcher.Match(tc.inputImage1, tc.inputImage2))
			assertImagesEqualWithMessage(t, tc.expectedSobelOutput, matcher.SobelOutput(), "sobel output")
			assertImagesEqualWithMessage(t, tc.expectedImage1WithEdgesRemoved, matcher.ExpectedImageWithEdgesRemoved(), "image1 with edges removed")
			assertImagesEqualWithMessage(t, tc.expectedImage2WithEdgesRemoved, matcher.ActualImageWithEdgesRemoved(), "image2 with edges removed")
			assert.Equal(t, tc.expectedNumDifferentPixels, matcher.Matcher.NumDifferentPixels())
			assert.Equal(t, tc.expectedMaxPixelDelta, matcher.Matcher.MaxPixelDelta())
		})
	}
}

func TestMatcher_Match_DifferentSizeImages_ReturnsFalse(t *testing.T) {

	smallImage := text.MustToNRGBA(`! SKTEXTSIMPLE
	7 7
	0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00`)

	largeImage := text.MustToNRGBA(`! SKTEXTSIMPLE
	8 8
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`)

	matcher := Matcher{
		Matcher: fuzzy.Matcher{
			MaxDifferentPixels:  1000,
			PixelDeltaThreshold: 10,
		},
		EdgeThreshold: 0xFF,
	}

	assert.False(t, matcher.Match(smallImage, largeImage))
	assert.False(t, matcher.Match(largeImage, smallImage))
}

// TestSobel_Success tests the sobel() function using the canonical image1 and image1Sobel images
// used throughout this file.
func TestSobel_Success(t *testing.T) {
	assertImagesEqual(t, text.MustToGray(image1Sobel), sobel(text.MustToGray(image1)))
}

// TestSobel_SmallImages_Success tests various edge cases involving small images.
//
// Notes:
//   - In practice the sobel() function will usually be invoked with much larger images.
//   - While these edge cases are unlikely to occur in practice, they exposed a number of bugs during
//     development (e.g. indexing and off-by-one errors). Those same bugs could be exposed with
//     larger, more realistic images, but using small images makes it easier to debug failing tests.
func TestSobel_SmallImages_Success(t *testing.T) {

	tests := []struct {
		name           string
		input          *image.Gray
		expectedOutput *image.Gray
	}{
		{
			name: "empty image, returns empty image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			0 0`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			0 0`),
		},
		{
			name: "1x1 black image, returns 1x1 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			1 1
			0x00`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			1 1
			0x00`),
		},
		{
			name: "1x1 white image, returns 1x1 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			1 1
			0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			1 1
			0x00`),
		},
		{
			name: "2x2 black image, returns 2x2 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			2 2
			0x00 0x00
			0x00 0x00`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			2 2
			0x00 0x00
			0x00 0x00`),
		},
		{
			name: "2x2 white image, returns 2x2 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			2 2
			0xFF 0xFF
			0xFF 0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			2 2
			0x00 0x00
			0x00 0x00`),
		},
		{
			name: "2x2 image with an edge, returns 2x2 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			2 2
			0x00 0xFF
			0x00 0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			2 2
			0x00 0x00
			0x00 0x00`),
		},
		{
			name: "3x3 black image, returns 3x3 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			3 3
			0x00 0x00 0x00
			0x00 0x00 0x00
			0x00 0x00 0x00`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			3 3
			0x00 0x00 0x00
			0x00 0x00 0x00
			0x00 0x00 0x00`),
		},
		{
			name: "3x3 white image, returns 3x3 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			3 3
			0xFF 0xFF 0xFF
			0xFF 0xFF 0xFF
			0xFF 0xFF 0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			3 3
			0x00 0x00 0x00
			0x00 0x00 0x00
			0x00 0x00 0x00`),
		},
		{
			name: "3x3 image with an edge, returns 3x3 image with one high intensity pixel",
			input: text.MustToGray(`! SKTEXTSIMPLE
			3 3
			0x00 0xFF 0xFF
			0x00 0xFF 0xFF
			0x00 0xFF 0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			3 3
			0x00 0x00 0x00
			0x00 0xFF 0x00
			0x00 0x00 0x00`),
		},
		{
			name: "4x4 black image, returns 4x4 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			4 4
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			4 4
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00`),
		},
		{
			name: "4x4 white image, returns 4x4 black image",
			input: text.MustToGray(`! SKTEXTSIMPLE
			4 4
			0xFF 0xFF 0xFF 0xFF
			0xFF 0xFF 0xFF 0xFF
			0xFF 0xFF 0xFF 0xFF
			0xFF 0xFF 0xFF 0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			4 4
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00
			0x00 0x00 0x00 0x00`),
		},
		{
			name: "4x4 image with an edge, returns 4x4 image with 4 high intensity pixels",
			input: text.MustToGray(`! SKTEXTSIMPLE
			4 4
			0x00 0x00 0xFF 0xFF
			0x00 0x00 0xFF 0xFF
			0x00 0x00 0xFF 0xFF
			0x00 0x00 0xFF 0xFF`),
			expectedOutput: text.MustToGray(`! SKTEXTSIMPLE
			4 4
			0x00 0x00 0x00 0x00
			0x00 0xFF 0xFF 0x00
			0x00 0xFF 0xFF 0x00
			0x00 0x00 0x00 0x00`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertImagesEqual(t, tc.expectedOutput, sobel(tc.input))
		})
	}
}

func TestSobel_EdgesAtVariousAngles_Success(t *testing.T) {

	input0Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x44 0x44 0x44 0x44 0x44`)

	expectedOutput0Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0xFF 0xFF 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`)

	input30Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x0E 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x00 0x36 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x0E 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x36 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x0E 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x36 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x0E 0x44 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x36 0x44 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x0E 0x44 0x44 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x36 0x44 0x44 0x44 0x44 0x44 0x44 0x44`)

	expectedOutput30Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x13 0xCE 0xFF 0x62 0x00 0x00
	0x00 0x00 0x00 0x00 0x62 0xFF 0xCE 0x13 0x00 0x00
	0x00 0x00 0x00 0x13 0xCE 0xFF 0x62 0x00 0x00 0x00
	0x00 0x00 0x00 0x62 0xFF 0xCE 0x13 0x00 0x00 0x00
	0x00 0x00 0x13 0xCE 0xFF 0x62 0x00 0x00 0x00 0x00
	0x00 0x00 0x62 0xFF 0xCE 0x13 0x00 0x00 0x00 0x00
	0x00 0x13 0xCE 0xFF 0x62 0x00 0x00 0x00 0x00 0x00
	0x00 0x62 0xFF 0xCE 0x13 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`)

	input45Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x05 0x3F
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x05 0x3F 0x44
	0x00 0x00 0x00 0x00 0x00 0x00 0x05 0x3F 0x44 0x44
	0x00 0x00 0x00 0x00 0x00 0x05 0x3F 0x44 0x44 0x44
	0x00 0x00 0x00 0x00 0x05 0x3F 0x44 0x44 0x44 0x44
	0x00 0x00 0x00 0x05 0x3F 0x44 0x44 0x44 0x44 0x44
	0x00 0x00 0x05 0x3F 0x44 0x44 0x44 0x44 0x44 0x44
	0x00 0x05 0x3F 0x44 0x44 0x44 0x44 0x44 0x44 0x44
	0x05 0x3F 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44
	0x3F 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44`)

	expectedOutput45Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x07 0x67 0xFF 0xFF 0x00
	0x00 0x00 0x00 0x00 0x07 0x67 0xFF 0xFF 0x67 0x00
	0x00 0x00 0x00 0x07 0x67 0xFF 0xFF 0x67 0x07 0x00
	0x00 0x00 0x07 0x67 0xFF 0xFF 0x67 0x07 0x00 0x00
	0x00 0x07 0x67 0xFF 0xFF 0x67 0x07 0x00 0x00 0x00
	0x00 0x67 0xFF 0xFF 0x67 0x07 0x00 0x00 0x00 0x00
	0x00 0xFF 0xFF 0x67 0x07 0x00 0x00 0x00 0x00 0x00
	0x00 0xFF 0x67 0x07 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`)

	input60Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x0E 0x36
	0x00 0x00 0x00 0x00 0x00 0x00 0x0E 0x36 0x44 0x44
	0x00 0x00 0x00 0x00 0x0E 0x36 0x44 0x44 0x44 0x44
	0x00 0x00 0x0E 0x36 0x44 0x44 0x44 0x44 0x44 0x44
	0x0E 0x36 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44
	0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44
	0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44
	0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44 0x44`)

	expectedOutput60Degrees := text.MustToGray(`! SKTEXTSIMPLE
	10 10
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x13 0x62 0x00
	0x00 0x00 0x00 0x00 0x00 0x13 0x62 0xCE 0xFF 0x00
	0x00 0x00 0x00 0x13 0x62 0xCE 0xFF 0xFF 0xCE 0x00
	0x00 0x13 0x62 0xCE 0xFF 0xFF 0xCE 0x62 0x13 0x00
	0x00 0xCE 0xFF 0xFF 0xCE 0x62 0x13 0x00 0x00 0x00
	0x00 0xFF 0xCE 0x62 0x13 0x00 0x00 0x00 0x00 0x00
	0x00 0x62 0x13 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
	0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`)

	test := func(name string, input, expectedOutput *image.Gray) {
		t.Run(name, func(t *testing.T) {
			assertImagesEqual(t, expectedOutput, sobel(input))
		})
	}

	test("0 degrees", input0Degrees, expectedOutput0Degrees)
	test("30 degrees", input30Degrees, expectedOutput30Degrees)
	test("45 degrees", input45Degrees, expectedOutput45Degrees)
	test("60 degrees", input60Degrees, expectedOutput60Degrees)
}

func TestSobel_GoldenImage_Success(t *testing.T) {

	// Attribution for the test/input.png image used below:
	//
	//   Author: Simpsons contributor.
	//   License: CC BY-SA (https://creativecommons.org/licenses/by-sa/3.0).
	//   Source: https://en.wikipedia.org/wiki/File:Valve_original_%281%29.PNG.
	//   Modifications: PNG image was recoded using Golang's png.Decode() and png.Encode().

	input := readPngAsGray(t, "test/input.png")
	expectedOutput := readPngAsGray(t, "test/sobel-expected-output.png")
	assert.Equal(t, expectedOutput, sobel(input))
}

// TestZeroOutEdges_Success tests function zeroOutEdges() against image1 and image2 using the edges
// from image1 (i.e. image1Sobel) in both cases, and a variety of edge thresholds.
func TestZeroOutEdges_Success(t *testing.T) {

	test := func(name, expectedOutput, input string, edgeThreshold uint8) {
		t.Run(name, func(t *testing.T) {
			// All tests use the edges from image1 (i.e. image1Sobel).
			actualOutput := zeroOutEdges(text.MustToNRGBA(input), text.MustToGray(image1Sobel), edgeThreshold)
			assertImagesEqual(t, text.MustToNRGBA(expectedOutput), actualOutput)
		})
	}

	// Test against image1.
	test("image1 with edge threshold 0xFF, no pixels zeroed out", image1, image1, 0xFF)
	test("image1 with edge threshold 0xAA, some pixels zeroed out", image1NoEdgesAbove0xAA, image1, 0xAA)
	test("image1 with edge threshold 0x66, some pixels zeroed out", image1NoEdgesAbove0x66, image1, 0x66)
	test("image1 with edge threshold 0x00, some pixels zeroed out", image1NoEdgesAbove0x00, image1, 0x00)

	// Test against image2.
	test("image2 with edge threshold 0xFF, no pixels zeroed out", image2, image2, 0xFF)
	test("image2 with edge threshold 0xAA, some pixels zeroed out", image2NoEdgesAbove0xAA, image2, 0xAA)
	test("image2 with edge threshold 0x66, some pixels zeroed out", image2NoEdgesAbove0x66, image2, 0x66)
	test("image2 with edge threshold 0x00, some pixels zeroed out", image2NoEdgesAbove0x00, image2, 0x00)
}

// TestZeroOutEdges_SmallImages_Success tests various edge cases involving small images.
func TestZeroOutEdges_SmallImages_Success(t *testing.T) {

	tests := []struct {
		name           string
		input          image.Image
		edges          *image.Gray
		edgeThreshold  uint8
		expectedOutput image.Image
	}{
		{
			name: "empty image, returns empty image",
			input: text.MustToNRGBA(`! SKTEXTSIMPLE
			0 0`),
			edges: text.MustToGray(`! SKTEXTSIMPLE
			0 0`),
			edgeThreshold: 0,
			expectedOutput: text.MustToNRGBA(`! SKTEXTSIMPLE
			0 0`),
		},
		{
			name: "1x1 image with pixel below threshold, returns original image",
			input: text.MustToNRGBA(`! SKTEXTSIMPLE
			1 1
			0xAABBCCFF`),
			edges: text.MustToGray(`! SKTEXTSIMPLE
			1 1
			0x55`),
			edgeThreshold: 0xBB,
			expectedOutput: text.MustToNRGBA(`! SKTEXTSIMPLE
			1 1
			0xAABBCCFF`),
		},
		{
			name: "1x1 image with pixel above threshold, returns black image",
			input: text.MustToNRGBA(`! SKTEXTSIMPLE
			1 1
			0xAABBCCFF`),
			edges: text.MustToGray(`! SKTEXTSIMPLE
			1 1
			0xCC`),
			edgeThreshold: 0xBB,
			expectedOutput: text.MustToNRGBA(`! SKTEXTSIMPLE
			1 1
			0x000000FF`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertImagesEqual(t, tc.expectedOutput, zeroOutEdges(tc.input, tc.edges, tc.edgeThreshold))
		})
	}
}

func TestZeroOutEdges_GoldenImage_Success(t *testing.T) {

	input := readPng(t, "test/input.png")
	edges := readPngAsGray(t, "test/sobel-expected-output.png")
	expectedOutput := readPng(t, "test/zero-out-edges-expected-output.png")
	assert.Equal(t, expectedOutput, zeroOutEdges(input, edges, 0x55))
}

func TestZeroOutEdges_InputAndEdgesImagesHaveDifferentBounds_Panics(t *testing.T) {

	assert.Panics(t, func() {
		img := text.MustToNRGBA(`! SKTEXTSIMPLE
		2 1
		0x00 0x00`)
		edges := text.MustToGray(`! SKTEXTSIMPLE
		1 1
		0x00`)
		zeroOutEdges(img, edges, 0)
	})
}

// image1 is a grayscale image with a diagonal, antialiased edge from 0x44 to 0x88.
const image1 = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x44 0x44 0x49 0x83 0x88
0x44 0x44 0x44 0x44 0x49 0x83 0x88 0x88
0x44 0x44 0x44 0x49 0x83 0x88 0x88 0x88
0x44 0x44 0x49 0x83 0x88 0x88 0x88 0x88
0x44 0x49 0x83 0x88 0x88 0x88 0x88 0x88
0x49 0x83 0x88 0x88 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image1Sobel is a grayscale image with the result of applying the Sobel operator to each
// non-border pixel in image1. Border pixels are set to 0.
const image1Sobel = `! SKTEXTSIMPLE
8 8
0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00
0x00 0x00 0x00 0x07 0x67 0xFF 0xFF 0x00
0x00 0x00 0x07 0x67 0xFF 0xFF 0x67 0x00
0x00 0x07 0x67 0xFF 0xFF 0x67 0x07 0x00
0x00 0x67 0xFF 0xFF 0x67 0x07 0x00 0x00
0x00 0xFF 0xFF 0x67 0x07 0x00 0x00 0x00
0x00 0xFF 0x67 0x07 0x00 0x00 0x00 0x00
0x00 0x00 0x00 0x00 0x00 0x00 0x00 0x00`

// image1NoEdgesAbove0xAA is the result of zeroing out all pixels in image1 with a Sobel operator
// value above 0xAA.
const image1NoEdgesAbove0xAA = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x44 0x44 0x00 0x00 0x88
0x44 0x44 0x44 0x44 0x00 0x00 0x88 0x88
0x44 0x44 0x44 0x00 0x00 0x88 0x88 0x88
0x44 0x44 0x00 0x00 0x88 0x88 0x88 0x88
0x44 0x00 0x00 0x88 0x88 0x88 0x88 0x88
0x49 0x00 0x88 0x88 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image1NoEdgesAbove0x66 is the result of zeroing out all pixels in image1 with a Sobel operator
// value above 0x66.
const image1NoEdgesAbove0x66 = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x44 0x00 0x00 0x00 0x88
0x44 0x44 0x44 0x00 0x00 0x00 0x00 0x88
0x44 0x44 0x00 0x00 0x00 0x00 0x88 0x88
0x44 0x00 0x00 0x00 0x00 0x88 0x88 0x88
0x44 0x00 0x00 0x00 0x88 0x88 0x88 0x88
0x49 0x00 0x00 0x88 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image1NoEdgesAbove0x00 is the result of zeroing out all pixels in image1 with a Sobel operator
// value above 0.
const image1NoEdgesAbove0x00 = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x00 0x00 0x00 0x00 0x88
0x44 0x44 0x00 0x00 0x00 0x00 0x00 0x88
0x44 0x00 0x00 0x00 0x00 0x00 0x00 0x88
0x44 0x00 0x00 0x00 0x00 0x00 0x88 0x88
0x44 0x00 0x00 0x00 0x00 0x88 0x88 0x88
0x49 0x00 0x00 0x00 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image2 is identical to image1 except for some antialiasing differences.
//
// It differs from image1 by 10 pixels, with a maximum per-channel delta sum of 36.
const image2 = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x44 0x44 0x49 0x83 0x88
0x44 0x44 0x47 0x49 0x55 0x83 0x88 0x88
0x44 0x44 0x49 0x4D 0x7F 0x87 0x88 0x88
0x44 0x44 0x55 0x7F 0x88 0x88 0x88 0x88
0x44 0x49 0x83 0x87 0x88 0x88 0x88 0x88
0x49 0x83 0x88 0x88 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image2NoEdgesAbove0xAA is the result of zeroing out all pixels in image2 where the Sobel
// operator value for the corresponding pixel in image1 is above 0xAA.
//
// It differs from image1NoEdgesAbove0xAA by 5 pixels, with a maximum per-channel delta sum of 15.
const image2NoEdgesAbove0xAA = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x44 0x44 0x00 0x00 0x88
0x44 0x44 0x47 0x49 0x00 0x00 0x88 0x88
0x44 0x44 0x49 0x00 0x00 0x87 0x88 0x88
0x44 0x44 0x00 0x00 0x88 0x88 0x88 0x88
0x44 0x00 0x00 0x87 0x88 0x88 0x88 0x88
0x49 0x00 0x88 0x88 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image2NoEdgesAbove0x66 is the result of zeroing out all pixels in image2 where the Sobel
// operator value for the corresponding pixel in image1 is above 0x66.
//
// It differs from image1NoEdgesAbove0x66 by 1 pixel, with a maximum per-channel delta sum of 9.
const image2NoEdgesAbove0x66 = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x44 0x00 0x00 0x00 0x88
0x44 0x44 0x47 0x00 0x00 0x00 0x00 0x88
0x44 0x44 0x00 0x00 0x00 0x00 0x88 0x88
0x44 0x00 0x00 0x00 0x00 0x88 0x88 0x88
0x44 0x00 0x00 0x00 0x88 0x88 0x88 0x88
0x49 0x00 0x00 0x88 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// image2NoEdgesAbove0x00 is the result of zeroing out all pixels in image2 where the Sobel
// operator value for the corresponding pixel in image1 is above 0.
//
// It is identical to image1NoEdgesAbove0x00.
const image2NoEdgesAbove0x00 = `! SKTEXTSIMPLE
8 8
0x44 0x44 0x44 0x44 0x44 0x44 0x49 0x83
0x44 0x44 0x44 0x00 0x00 0x00 0x00 0x88
0x44 0x44 0x00 0x00 0x00 0x00 0x00 0x88
0x44 0x00 0x00 0x00 0x00 0x00 0x00 0x88
0x44 0x00 0x00 0x00 0x00 0x00 0x88 0x88
0x44 0x00 0x00 0x00 0x00 0x88 0x88 0x88
0x49 0x00 0x00 0x00 0x88 0x88 0x88 0x88
0x83 0x88 0x88 0x88 0x88 0x88 0x88 0x88`

// assertImagesEqual asserts that the two given images are equal, and prints out the actual image
// encoded as SKTEXT if the assertion is false.
func assertImagesEqual(t *testing.T, expected, actual image.Image) {
	assertImagesEqualWithMessage(t, expected, actual, "")
}

// assertImagesEqualWithMessage asserts that the two given images are equal, and prints out the
// actual image encoded as SKTEXT if the assertion is false, along with the given message if it is
// not empty.
func assertImagesEqualWithMessage(t *testing.T, expected, actual image.Image, msg string) {
	// No need for newline if message is empty.
	if msg != "" {
		msg = msg + "\n"
	}

	assert.Equal(t, expected, actual, fmt.Sprintf("%sSKTEXT-encoded expected output:\n%s\nSKTEXT-encoded actual output:\n%s", msg, imageToText(t, expected), imageToText(t, actual)))
}

// imageToText returns the given image as an SKTEXT-encoded string.
func imageToText(t *testing.T, img image.Image) string {
	// Convert image to NRGBA.
	nrgbaImg := image.NewNRGBA(img.Bounds())
	draw.Draw(nrgbaImg, img.Bounds(), img, img.Bounds().Min, draw.Src)

	// Encode and return image as SKTEXTSIMPLE.
	buf := &bytes.Buffer{}
	err := text.Encode(buf, nrgbaImg)
	require.NoError(t, err)
	return buf.String()
}

// readPngAsGray reads a PNG image from the file system, converts it to grayscale and returns it as
// an *image.Gray.
func readPngAsGray(t *testing.T, filename string) *image.Gray {
	return imageToGray(readPng(t, filename))
}

// readPng reads a PNG image from the file system and returns it as an *image.NRGBA.
func readPng(t *testing.T, filename string) *image.NRGBA {
	// Read image.
	imgBytes, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	// Decode image.
	img, err := png.Decode(bytes.NewReader(imgBytes))
	require.NoError(t, err)

	// Convert to NRGBA.
	nrgbaImg := image.NewNRGBA(img.Bounds())
	draw.Draw(nrgbaImg, img.Bounds(), img, img.Bounds().Min, draw.Src)

	return nrgbaImg
}
