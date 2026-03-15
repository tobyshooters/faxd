package source

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func printFile(path string, cfg Config) error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
		dithered, err := ditherImage(path)
		if err != nil {
			return fmt.Errorf("dither: %w", err)
		}
		defer os.Remove(dithered)
		path = dithered
	}

	args := []string{"-o", "print-scaling=auto"}
	if cfg.Scaling > 0 {
		args = append(args, "-o", fmt.Sprintf("scaling=%d", cfg.Scaling))
	}
	if cfg.Monochrome {
		args = append(args, "-o", "ColorModel=Gray")
	}
	args = append(args, path)

	return exec.Command("lp", args...).Run()
}

func ditherImage(path string) (string, error) {
	out := path + ".dithered.png"
	cmd := exec.Command("convert", path,
		"-resize", "800x800>",
		"-colorspace", "Gray",
		"-brightness-contrast", "10x30",
		"-dither", "FloydSteinberg",
		"-monochrome",
		out,
	)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out, nil
}
