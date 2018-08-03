package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/pkg/reexec"
	"github.com/fuweid/bench-fmountat/fmountat"
	"github.com/fuweid/bench-fmountat/rmountat"
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

// Benchmark_FMountat_16Layer runs the FMountat to mount 16 lower layers.
func Benchmark_FMountat_16Layer(b *testing.B) {
	benchmarkMountNLayer(b, 16, false)
}

// Benchmark_FMountat_32Layer runs the FMountat to mount 32 lower layers.
func Benchmark_FMountat_32Layer(b *testing.B) {
	benchmarkMountNLayer(b, 32, false)
}

// Benchmark_FMountat_64Layer runs the FMountat to mount 64 lower layers.
func Benchmark_FMountat_64Layer(b *testing.B) {
	benchmarkMountNLayer(b, 64, false)
}

// Benchmark_FMountat_128Layer runs the FMountat to mount 128 lower layers.
func Benchmark_FMountat_128Layer(b *testing.B) {
	benchmarkMountNLayer(b, 128, false)
}

// Benchmark_RMountat_16Layer runs the RMountat to mount 16 lower layers.
func Benchmark_RMountat_16Layer(b *testing.B) {
	benchmarkMountNLayer(b, 16, true)
}

// Benchmark_RMountat_32Layer runs the RMountat to mount 32 lower layers.
func Benchmark_RMountat_32Layer(b *testing.B) {
	benchmarkMountNLayer(b, 32, true)
}

// Benchmark_RMountat_64Layer runs the RMountat to mount 64 lower layers.
func Benchmark_RMountat_64Layer(b *testing.B) {
	benchmarkMountNLayer(b, 64, true)
}

// Benchmark_RMountat_128Layer runs the RMountat to mount 128 lower layers.
func Benchmark_RMountat_128Layer(b *testing.B) {
	benchmarkMountNLayer(b, 128, true)
}

func benchmarkMountNLayer(b *testing.B, n int, isReexec bool) {
	b.StopTimer()

	opt, chdir, cleanup, err := prepareOverlayNLayersData(n)
	if err != nil {
		b.Fatalf("failed to prepare mount data for overlay N layers: %v", err)
	}
	defer cleanup()

	// open chdir for FMountat
	cf, err := os.Open(chdir)
	if err != nil {
		b.Fatalf("failed to open chdir: %v", err)
	}
	defer cf.Close()

	for i := 0; i < b.N; i++ {
		b.StartTimer()

		if isReexec {
			// run for the re-exec mountat
			if err := rmountat.RMountat(chdir, opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
				b.Fatal(err)
			}
		} else {
			// run for the fork mountat
			if err := fmountat.FMountat(cf.Fd(), opt.source, opt.target, opt.fstype, opt.flags, opt.data); err != nil {
				b.Fatal(err)
			}
		}

		b.StopTimer()
		umount(b, opt.target)
	}
}
