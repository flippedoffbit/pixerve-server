package encoder

import (
	"context"
	"log"
	"os/exec"
)

// EncodeFunc is the function signature for any encoder
type EncodeFunc func(ctx context.Context, input, output string, opts EncodeOptions) error

type EncodeOptions struct {
	Width, Height int
	Quality       int
	Speed         int
}

// Registry maps format name → encoder function
var Registry = map[string]EncodeFunc{}

// Register adds encoder if the underlying command exists, logs status
func Register(format string, cmdName string, fn EncodeFunc) {
	if _, err := exec.LookPath(cmdName); err != nil {
		log.Printf("encoder [%s] skipped: command '%s' not found in PATH", format, cmdName)
		return
	}
	Registry[format] = fn
	log.Printf("encoder [%s] registered (command: %s)", format, cmdName)
}

// Lookup encoder by format
func Get(format string) (EncodeFunc, bool) {
	fn, ok := Registry[format]
	return fn, ok
}

// Explicit defaults registration
func RegisterDefaults() {
	Register("jpg", "magick", EncodeJPG)
	Register("png", "magick", EncodePNG)
	Register("webp", "cwebp", EncodeWebP)
	Register("avif", "avifenc", EncodeAVIF)
}
