package encoder

import (
	"context"
	"fmt"
	"os/exec"
)

func EncodeAVIF(ctx context.Context, in, out string, o EncodeOptions) error {
	args := []string{
		"--min", fmt.Sprint(o.Quality),
		"--max", fmt.Sprint(o.Quality),
		"--speed", fmt.Sprint(o.Speed),
		"--resize", fmt.Sprintf("%dx%d", o.Width, o.Height),
		in, out,
	}
	cmd := exec.CommandContext(ctx, "avifenc", args...)
	return cmd.Run()
}
