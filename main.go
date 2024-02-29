package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"fyne.io/systray"
	"github.com/adrg/xdg"
	"github.com/jessevdk/go-flags"
	"github.com/relvacode/pulseclip/internal/capture"
	"github.com/relvacode/pulseclip/internal/notification"
	"github.com/relvacode/pulseclip/resources"
)

type Duration struct {
	time.Duration
}

func (f *Duration) UnmarshalFlag(value string) error {
	t, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	f.Duration = t
	return nil
}

type Options struct {
	CaptureBuffer Duration `short:"d" long:"capture-buffer" default:"30s" description:"Capture the last N duration"`
	ClipPath      string   `long:"clip-path" description:"encode clips to this directory. Defaults to $XDG_MUSIC_DIR/PulseClips"`
	Notify        bool     `long:"notify" description:"Enable DBus notifications"`
}

const (
	signalClip = syscall.SIGUSR1
	signalQuit = syscall.SIGTERM
)

func Main() error {
	var opts Options
	_, err := flags.NewParser(&opts, flags.HelpFlag).Parse()
	if err != nil {
		return err
	}

	clipPath := opts.ClipPath
	if clipPath == "" {
		clipPath = filepath.Join(xdg.UserDirs.Music, "PulseClips")
	}

	err = os.MkdirAll(clipPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create clip directory: %w", err)
	}

	var notify notification.Notifier = notification.Null{}
	if opts.Notify {
		notify = notification.DBus()
	}

	defer notify.Close()

	log.Printf("Start recording using a %s buffer", opts.CaptureBuffer)

	rec, err := capture.New(clipPath, notify, opts.CaptureBuffer.Duration)
	if err != nil {
		return err
	}

	defer rec.Close()

	signals := make(chan os.Signal, 3)
	defer close(signals)

	signal.Notify(signals, os.Interrupt, signalQuit, signalClip)
	defer signal.Stop(signals)

	go func() {
		for sig := range signals {
			switch sig {
			case signalClip:
				rec.Signal(capture.Capture{})
			case signalQuit:
				rec.Signal(capture.Quit{})
			}
		}
	}()

	go systray.Run(
		func() {
			systray.SetTemplateIcon(resources.IconActive, resources.IconActive)
			systray.SetTitle("Pulse Clip")

			miClip := systray.AddMenuItem("Save Clip", "Save a new clip")
			systray.AddSeparator()
			miQuit := systray.AddMenuItem("Quit", "Quit")

			// Dispatch systray events
			go func() {
				for {
					select {
					case <-miClip.ClickedCh:
						rec.Signal(capture.Capture{})
					case <-miQuit.ClickedCh:
						rec.Signal(capture.Quit{})
					}
				}
			}()
		},
		func() {
		},
	)

	return rec.Start()
}

func main() {
	err := Main()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
