package capture

import (
	"bytes"
	"fmt"
	"github.com/viert/go-lame"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func encode(path string, buffer Buffer) error {
	w, err := os.Create(path)
	if err != nil {
		return err
	}

	var isSafeClosed bool
	defer func() {
		if !isSafeClosed {
			_ = w.Close()
			_ = os.Remove(path)
		}
	}()

	enc := lame.NewEncoder(w)
	_ = enc.SetNumChannels(pulseChannels)
	_ = enc.SetInSamplerate(pulseSampleRate)
	_ = enc.SetQuality(2)

	_, err = bytes.NewReader(buffer).WriteTo(enc)
	if err != nil {
		return fmt.Errorf("failed to encode mp3 data: %w", err)
	}

	enc.Close()

	err = w.Close()
	isSafeClosed = true
	if err != nil {
		return fmt.Errorf("failed to save clip: %w", err)
	}

	return nil
}

var clipIncrRegex = regexp.MustCompile(`^pulseclip-([0-9]+)\.mp3$`)

func writeClip(dirPath string, buffer Buffer) (string, error) {
	currentDirEnt, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}

	var maxCounterVal int
	for _, f := range currentDirEnt {
		match := clipIncrRegex.FindStringSubmatch(f.Name())
		if match == nil {
			continue
		}

		counterVal, _ := strconv.Atoi(match[1])
		maxCounterVal = max(counterVal, maxCounterVal)
	}

	var (
		filename = fmt.Sprintf("pulseclip-%03d.mp3", maxCounterVal+1)
		fullPath = filepath.Join(dirPath, filename)
	)

	err = encode(fullPath, buffer)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}
