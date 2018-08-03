package rmountat

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"golang.org/x/sys/unix"
)

func init() {
	reexec.Register("rmountat", rmountatMain)
}

type mountOption struct {
	Source string
	Target string
	FsType string
	Flags  uintptr
	Data   string
}

func RMountat(chdir string, source, target, ftype string, flags uintptr, data string) error {
	opt := &mountOption{
		Source: source,
		Target: target,
		FsType: ftype,
		Flags:  flags,
		Data:   data,
	}

	cmd := reexec.Command("rmountat", chdir)
	w, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("rmountat error on stdin pipe: %v", err)
	}

	out := bytes.NewBuffer(nil)
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		w.Close()
		return fmt.Errorf("rmountat error on start cmd: %v", err)
	}

	//write the options to the pipe for the untar exec to read
	if err := json.NewEncoder(w).Encode(opt); err != nil {
		w.Close()
		return fmt.Errorf("rmountat json encode to pipe failed: %v", err)
	}

	w.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("rmountat error: %v: output: %v", err, out)
	}
	return nil
}

func rmountatMain() {
	flag.Parse()

	var opt *mountOption

	if err := json.NewDecoder(os.Stdin).Decode(&opt); err != nil {
		fatal(err)
	}

	if err := os.Chdir(flag.Arg(0)); err != nil {
		fatal(err)
	}

	if err := unix.Mount(opt.Source, opt.Target, opt.FsType, opt.Flags, opt.Data); err != nil {
		fatal(err)
	}

	os.Exit(0)
}

func fatal(err error) {
	fmt.Fprint(os.Stderr, err)
	os.Exit(1)
}
