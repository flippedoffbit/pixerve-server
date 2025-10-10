package encoder

import (
	"context"
	"fmt"
	"os/exec"
)

func EncodeWebP(ctx context.Context, in, out string, o EncodeOptions) error {
	args := []string{
		"-q", fmt.Sprint(o.Quality),
		"-m", fmt.Sprint(o.Speed),
		"-resize", fmt.Sprint(o.Width), fmt.Sprint(o.Height),
		in, "-o", out,
	}
	cmd := exec.CommandContext(ctx, "cwebp", args...)
	return cmd.Run()
}
