package encoder

import (
	"context"
	"fmt"
	"os/exec"
)

// EncodeJPG encodes using ImageMagick
func EncodeJPG(ctx context.Context, in, out string, o EncodeOptions) error {
	return magickEncode(ctx, in, out, o, "jpg")
}

// EncodePNG encodes using ImageMagick
func EncodePNG(ctx context.Context, in, out string, o EncodeOptions) error {
	return magickEncode(ctx, in, out, o, "png")
}

// Shared helper for magick-based formats
func magickEncode(ctx context.Context, in, out string, o EncodeOptions, format string) error {
	args := []string{
		in,
		"-resize", fmt.Sprintf("%dx%d", o.Width, o.Height),
		"-quality", fmt.Sprint(o.Quality),
		fmt.Sprintf("%s:%s", format, out),
	}
	cmd := exec.CommandContext(ctx, "magick", args...)
	return cmd.Run()
}
