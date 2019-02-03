package main

import (
	"flag"
	"image"
	"image/color"
	"log"

	"github.com/disintegration/imaging"
)

var (
	backgroundBlurFactor  = flag.Uint("bg-blur-factor", 35, "Background blur factor")
	backgroundGammaFactor = flag.Float64("bg-gamma-factor", 0.9, "Background gamma factor")
	in                    = flag.String("in", "", "Input image name")
	lowerCropBegin        = flag.Uint("lower-crop-begin-compare", 2, "Number of pixels to skip before comparison starts")
	lowerCropHorizOffset  = flag.Uint("lower-crop-horizontal-offset", 10, "Offset from the left to compare for lower crop")
	lowerCropRgb          = flag.Uint("lower-crop-rgb", 61680, "R/G/B value 0...65535 to crop from the bottom")
	maxLowerResize        = flag.Uint("max-lower-resize", 200, "Maximum number of rows to crop from bottom")
	maxUpperResize        = flag.Uint("max-upper-resize", 200, "Maximum number of rows to crop from top")
	out                   = flag.String("out", "", "Output image name")
	targetHeight          = flag.Uint("target-height", 1080, "Target height")
	targetWidth           = flag.Uint("target-width", 1920, "Target width")
	upperCropBegin        = flag.Uint("upper-crop-begin-compare", 2, "Number of pixels to skip before comparison starts")
	upperCropHorizOffset  = flag.Uint("upper-crop-horizontal-offset", 10, "Offset from the left to compare for upper crop")
	upperCropRgb          = flag.Uint("upper-crop-rgb", 0, "R/G/B value 0...65535 to crop from the top")
)

func main() {
	flag.Parse()

	if *in == "" || *out == "" {
		flag.PrintDefaults()
		return
	}

	// Open a test image.
	src, err := imaging.Open(*in)
	if err != nil {
		log.Fatalf("failed to open image: %v", err)
	}

	// Shave off #efefef bottom for android screencaps
	bottom := uint(src.Bounds().Max.Y)
	for i := bottom - *lowerCropBegin; i >= bottom-(*maxLowerResize*2); i-- {
		r, g, b, _ := src.At(int(*lowerCropHorizOffset), int(i)).RGBA()
		//log.Printf("x = 10, y = %d, r = %d, g = %d, b = %d", i, r, g, b)
		bottom = i
		if r != uint32(*lowerCropRgb) || g != uint32(*lowerCropRgb) || b != uint32(*lowerCropRgb) {
			break
		}
	}
	log.Printf("Calculated bottom px at %d", bottom)

	// Shave off top, if it exists
	top := uint(0)
	for i := top + *upperCropBegin; i <= (*maxUpperResize * 2); i++ {
		r, g, b, _ := src.At(int(*upperCropHorizOffset), int(i)).RGBA()
		if r != uint32(*upperCropRgb) || g != uint32(*upperCropRgb) || b != uint32(*upperCropRgb) {
			break
		}
		top = i
	}

	if int(bottom)-src.Bounds().Max.Y > int(*maxLowerResize) {
		log.Printf("Difference was %d, resetting to bottom", int(bottom)-(src.Bounds().Max.Y))
		bottom = uint(src.Bounds().Max.Y)
	}
	if bottom != uint(src.Bounds().Max.Y) {
		log.Printf("Current bounds: %d, %d", src.Bounds().Max.X, src.Bounds().Max.Y)
		log.Printf("Resize detected; clipping bottom to %d", bottom)
		//src = imaging.Crop(src, image.Rect(0, src.Bounds().Max.X, 0, bottom))
		src = imaging.CropAnchor(src, src.Bounds().Max.X, int(bottom), imaging.TopLeft)
	}

	// Resize height to final keeping aspect ratio
	log.Printf("Current bounds: %d, %d", src.Bounds().Max.X, src.Bounds().Max.Y)
	log.Printf("Resizing height to %d, keeping aspect ratio", *targetHeight)
	src = imaging.Resize(src, 0, int(*targetHeight), imaging.Lanczos)

	// Resize background to final size, to hell with the aspect ratio
	log.Printf("Creating secondary blackground")
	blurBg := imaging.Resize(src, int(*targetWidth), int(*targetHeight), imaging.Lanczos)
	// Blur it
	log.Printf("Blurring background")
	blurBg = imaging.Blur(blurBg, float64(*backgroundBlurFactor))
	log.Printf("Setting gamma a little lower")
	blurBg = imaging.AdjustGamma(blurBg, float64(*backgroundGammaFactor))

	// Create new image
	log.Printf("Compositing image")
	dst := imaging.New(int(*targetWidth), int(*targetHeight), color.NRGBA{0, 0, 0, 0})
	dst = imaging.Paste(dst, blurBg, image.Pt(0, 0))
	dst = imaging.Paste(dst, src, image.Pt((int(*targetWidth)-src.Bounds().Max.X-1)/2, 0))

	// Save the resulting image as JPEG.
	err = imaging.Save(dst, *out)
	if err != nil {
		log.Fatalf("failed to save image: %v", err)
	}
}
