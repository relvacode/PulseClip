package capture

type Signal interface {
	signal()
}

// Quit issues a signal to quit recording
type Quit struct{}

func (Quit) signal() {}

// Capture issues a signal to immediately capture the current buffer
type Capture struct{}

func (Capture) signal() {}
