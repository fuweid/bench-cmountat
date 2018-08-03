package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/fuweid/bench-fmountat/fmountat"
	"github.com/fuweid/bench-fmountat/rmountat"
	"golang.org/x/sys/unix"
)

// mountOpt wraps mount syscall arguments.
type mountOpt struct {
	source string
	target string
	fstype string
	flags  uintptr
	data   string
}

// cleanupFn is used to present any clean up action.
type cleanupFn func() error

// the following format is used in the initOverlayNLowerLayer.
var (
	fmtContent  = "Hi, I'm No.%d Layer!"
	fmtFileName = "commit-%d"
)

// initOverlayNLowerLayer creates n folders which contains named commit-x file.
func initOverlayNLowerLayer(rootdir string, n int) ([]string, error) {
	dirs := make([]string, n)
	for i := range dirs {
		subdir := fmt.Sprintf("%d", i+1)
		filename := fmt.Sprintf(fmtFileName, i+1)
		content := fmt.Sprintf(fmtContent, i+1)

		a := fstest.Apply(
			fstest.CreateDir(subdir, 0644),
			fstest.CreateFile(filepath.Join(subdir, filename), []byte(content), 0644),
		)

		if err := a.Apply(rootdir); err != nil {
			return nil, err
		}
		dirs[i] = subdir
	}
	return dirs, nil
}

// prepareOverlayNLayersData prepares the benchmark test case data.
func prepareOverlayNLayersData(n int) (opt *mountOpt, chdir string, cleanup cleanupFn, err error) {
	var (
		root, target        string
		work, upper, common string
		lowerdirs           []string
	)

	// the level of root is like
	//
	//	${root}/target      => target mount point
	//	${root}/work        => work dir
	//	${root}/upper       => upper dir
	//	${root}/common      => common dir of lowedir
	//
	// each lower dir is named by the increasing integer from 1. It will
	// contains only one file named by "commit-X". X is the same to the
	// name of lower dir.
	root, err = ioutil.TempDir("", "fmountat-test-")
	if err != nil {
		return
	}

	target = filepath.Join(root, "target")
	if err = os.MkdirAll(target, 0777); err != nil {
		return
	}

	work = filepath.Join(root, "work")
	if err = os.MkdirAll(work, 0777); err != nil {
		return
	}

	upper = filepath.Join(root, "upper")
	if err = os.MkdirAll(upper, 0777); err != nil {
		return
	}

	common = filepath.Join(root, "common")
	if err = os.MkdirAll(common, 0777); err != nil {
		return
	}

	lowerdirs, err = initOverlayNLowerLayer(common, n)
	if err != nil {
		return
	}

	opt = &mountOpt{
		source: "overlay",
		target: target,
		fstype: "overlay",
		flags:  uintptr(0),
		data:   fmt.Sprintf("workdir=%s,upperdir=%s,lowerdir=%s", work, upper, strings.Join(lowerdirs, ":")),
	}

	cleanup = func() error {
		return os.RemoveAll(root)
	}

	chdir = common
	return
}

func Test_PrepareOverlayNLayersData(t *testing.T) {
	layerNum := 10

	t.Run("by FMountat", func(t *testing.T) {
		opt, chdir, cleanup, err := prepareOverlayNLayersData(layerNum)
		if err != nil {
			t.Fatalf("failed to prepare mount data for overlay N layers: %v", err)
		}
		defer cleanup()

		cf, err := os.Open(chdir)
		if err != nil {
			t.Fatalf("failed to open chdir: %v", err)
		}
		defer cf.Close()

		if err := fmountat.FMountat(cf.Fd(), opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
			t.Fatal(err)
		}

		testFileInMount(t, opt.target)
		umount(t, opt.target)
	})

	t.Run("by RMountat", func(t *testing.T) {
		opt, chdir, cleanup, err := prepareOverlayNLayersData(layerNum)
		if err != nil {
			t.Fatalf("failed to prepare mount data for overlay N layers: %v", err)
		}
		defer cleanup()

		if err := rmountat.RMountat(chdir, opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
			t.Fatal(err)
		}

		testFileInMount(t, opt.target)
		umount(t, opt.target)
	})
}

func testFileInMount(t testing.TB, target string) {
	filepath.Walk(target, func(path string, f os.FileInfo, err error) error {
		// skip the root dir
		if path == target {
			return nil
		}

		if f.IsDir() {
			t.Fatalf("unexpected dir in the target mount: %s", path)
		}

		var no int64
		if _, err := fmt.Sscanf(f.Name(), fmtFileName, &no); err != nil {
			t.Fatalf("expected to get filename named commit-x, but got: %s", path)
		}

		got, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to get the content from file(%s): %v", path, err)
		}

		expected := fmt.Sprintf(fmtContent, no)
		if string(got) != expected {
			t.Fatalf("expected content(%v), but got(%v)", expected, string(got))
		}

		return nil
	})
}

// umount unmounts the target folder.
func umount(t testing.TB, target string) {
	for i := 0; i < 50; i++ {
		if err := unix.Unmount(target, unix.MNT_DETACH); err != nil {
			switch err {
			case unix.EBUSY:
				time.Sleep(50 * time.Millisecond)
				continue
			case unix.EINVAL:
				return
			default:
				continue
			}
		}
	}
	t.Fatalf("failed to unmount target %s", target)
}
