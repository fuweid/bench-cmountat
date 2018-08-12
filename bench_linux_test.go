package main

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/docker/docker/pkg/reexec"
	"github.com/fuweid/bench-fmountat/fmountat"
	"github.com/fuweid/bench-fmountat/rmountat"
)

type benchType int

const (
	benchDirectly benchType = iota
	benchFMountat
	benchRMountat
)

// TestMain does init thing and run the cases.
func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}

	// require root user
	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "This test must be run by root.")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// Benchmark_Mount_16Layer runs the mount 16 lower layers directly.
func Benchmark_Mount_16Layer(b *testing.B) {
	benchmarkMountNLayer(b, 16, benchDirectly)
}

// Benchmark_Mount_32Layer runs the mount 32 lower layers directly.
func Benchmark_Mount_32Layer(b *testing.B) {
	benchmarkMountNLayer(b, 32, benchDirectly)
}

// Benchmark_Mount_64Layer runs the mount 64 lower layers directly.
func Benchmark_Mount_64Layer(b *testing.B) {
	benchmarkMountNLayer(b, 64, benchDirectly)
}

// Benchmark_Mount_128Layer runs the mount 128 lower layers directly.
func Benchmark_Mount_128Layer(b *testing.B) {
	benchmarkMountNLayer(b, 128, benchDirectly)
}

// Benchmark_FMountat_16Layer runs the FMountat to mount 16 lower layers.
func Benchmark_FMountat_16Layer(b *testing.B) {
	benchmarkMountNLayer(b, 16, benchFMountat)
}

// Benchmark_FMountat_32Layer runs the FMountat to mount 32 lower layers.
func Benchmark_FMountat_32Layer(b *testing.B) {
	benchmarkMountNLayer(b, 32, benchFMountat)
}

// Benchmark_FMountat_64Layer runs the FMountat to mount 64 lower layers.
func Benchmark_FMountat_64Layer(b *testing.B) {
	benchmarkMountNLayer(b, 64, benchFMountat)
}

// Benchmark_FMountat_128Layer runs the FMountat to mount 128 lower layers.
func Benchmark_FMountat_128Layer(b *testing.B) {
	benchmarkMountNLayer(b, 128, benchFMountat)
}

// Benchmark_RMountat_16Layer runs the RMountat to mount 16 lower layers.
func Benchmark_RMountat_16Layer(b *testing.B) {
	benchmarkMountNLayer(b, 16, benchRMountat)
}

// Benchmark_RMountat_32Layer runs the RMountat to mount 32 lower layers.
func Benchmark_RMountat_32Layer(b *testing.B) {
	benchmarkMountNLayer(b, 32, benchRMountat)
}

// Benchmark_RMountat_64Layer runs the RMountat to mount 64 lower layers.
func Benchmark_RMountat_64Layer(b *testing.B) {
	benchmarkMountNLayer(b, 64, benchRMountat)
}

// Benchmark_RMountat_128Layer runs the RMountat to mount 128 lower layers.
func Benchmark_RMountat_128Layer(b *testing.B) {
	benchmarkMountNLayer(b, 128, benchRMountat)
}

func benchmarkMountNLayer(b *testing.B, n int, typ benchType) {
	b.StopTimer()

	opt, chdir, cleanup, err := prepareOverlayNLayersData(n)
	if err != nil {
		b.Fatalf("failed to prepare mount data for overlay N layers: %v", err)
	}
	defer cleanup()

	// open chdir for benchFMountat
	cf, err := os.Open(chdir)
	if err != nil {
		b.Fatalf("failed to open chdir: %v", err)
	}
	defer cf.Close()

	// change current working dir for benchDirectly
	if typ == benchDirectly {
		oldWd, err := os.Getwd()
		if err != nil {
			b.Fatalf("failed to get current working dir: %v", err)
		}

		if err := os.Chdir(chdir); err != nil {
			b.Fatalf("failed to change current working dir before run directly mount: %v", err)
		}

		// recovery
		defer func() {
			if err := os.Chdir(oldWd); err != nil {
				b.Fatalf("failed to recover the old working dir: %v", err)
			}
		}()
	}

	for i := 0; i < b.N; i++ {
		b.StartTimer()

		switch typ {
		case benchDirectly:
			// run for the directly mount
			if err := syscall.Mount(opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
				b.Fatal(err)
			}
		case benchRMountat:
			// run for the re-exec mountat
			if err := rmountat.RMountat(chdir, opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
				b.Fatal(err)
			}
		case benchFMountat:
			// run for the fork mountat
			if err := fmountat.FMountat(cf.Fd(), opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
				b.Fatal(err)
			}
		default:
			b.Fatalf("not support bench testing type(%v)", typ)
		}

		b.StopTimer()
		umount(b, opt.target)
	}
}
