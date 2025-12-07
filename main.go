package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/gousb"
)

const (
	vid = 0x320f
	pid = 0x5055
)

const (
	dateOffset = 35
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("OK")
}
func run() error {
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := ctx.OpenDeviceWithVIDPID(vid, pid)
	if err != nil {

		return fmt.Errorf("could not open a device: %w", err)
	}
	if dev == nil {
		return errors.New("device not found")
	}
	defer dev.Close()
	if err := dev.SetAutoDetach(true); err != nil {
		return fmt.Errorf("error setting auto-detach: %w", err)
	}

	cfgNum, err := dev.ActiveConfigNum()
	if err != nil {
		return fmt.Errorf("error getting active configuration: %w", err)
	}

	cfg, err := dev.Config(cfgNum)
	if err != nil {
		return fmt.Errorf("error getting configuration %d: %w", cfgNum, err)
	}
	defer cfg.Close()

	intf, err := cfg.Interface(3, 0)
	if err != nil {
		return fmt.Errorf("error getting interface %d: %w", 3, err)
	}
	defer intf.Close()

	_ = intf
	return nil
}
