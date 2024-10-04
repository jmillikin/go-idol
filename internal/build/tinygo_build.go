package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	tinygo   = flag.String("tinygo", "", "")
	output   = flag.String("output", "", "")
	chdir    = flag.String("chdir", "", "")
	goSdkBin = flag.String("go-sdk-bin", "", "")
	wasmOpt  = flag.String("wasm-opt", "", "")
)

func main() {
	flag.Parse()
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tinygoArgs := []string{"build"}
	tinygoArgs = append(tinygoArgs, "-o="+filepath.Join(pwd, *output))
	tinygoArgs = append(tinygoArgs, flag.Args()...)

	cmd := exec.Command(filepath.Join(pwd, *tinygo), tinygoArgs...)
	cmd.Env = []string{
		"PATH=" + filepath.Join(pwd, *goSdkBin),
		"HOME=" + filepath.Join(os.Getenv("TMPDIR"), "tinygo-home"),
		"WASMOPT=" + filepath.Join(pwd, *wasmOpt),
	}
	cmd.Dir = filepath.Join(pwd, *chdir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
