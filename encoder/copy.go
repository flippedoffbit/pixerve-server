package encoder

import (
	"context"
	"io"
	"os"
	"pixerve/logger"
)

// EncodeCopy copies the input file to the output path without any encoding
// its made to handle cases where no encoding is needed but we want to keep the original file
// usually kept on for important images where loss of qualirty is not acceptable
func EncodeCopy(ctx context.Context, input, output string, opts EncodeOptions) error {
	src, err := os.Open(input)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(output)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	logger.Debugf("copied original file from %s to %s", input, output)
	return nil
}

// RegisterCopy registers the copy encoder (no command dependency)
func RegisterCopy() {
	Registry["copy"] = EncodeCopy
	logger.Debugf("encoder [copy] registered (no command required)")
}
