package kbd

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/gousb"
)

type USB struct {
	ctx    *gousb.Context
	device *gousb.Device
	cfg    *gousb.Config
	intf   *gousb.Interface
	inEp   *gousb.InEndpoint
	outEp  *gousb.OutEndpoint
}

func NewUSB(vid, pid gousb.ID) (*USB, error) {
	uctx := gousb.NewContext()

	// Open any device with a given VID/PID using a convenience function.
	dev, err := uctx.OpenDeviceWithVIDPID(vid, pid)
	if err != nil {
		uctx.Close()
		return nil, fmt.Errorf("could not open a device: %w", err)
	}
	if dev == nil {
		uctx.Close()
		return nil, errors.New("device not found")
	}

	if err := dev.SetAutoDetach(true); err != nil {
		dev.Close()
		uctx.Close()
		return nil, fmt.Errorf("error setting auto-detach: %w", err)
	}

	cfgNum, err := dev.ActiveConfigNum()
	if err != nil {
		dev.Close()
		uctx.Close()
		return nil, fmt.Errorf("error getting active configuration: %w", err)
	}

	cfg, err := dev.Config(cfgNum)
	if err != nil {
		dev.Close()
		uctx.Close()
		return nil, fmt.Errorf("error getting configuration %d: %w", cfgNum, err)
	}

	intf, err := cfg.Interface(kbIfaceNum, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		uctx.Close()
		return nil, fmt.Errorf("error getting interface %d: %w", kbIfaceNum, err)
	}

	set := intf.Setting
	var (
		inEp  *gousb.InEndpoint
		outEp *gousb.OutEndpoint
	)
	for _, ep := range set.Endpoints {
		var err error
		if ep.Direction == gousb.EndpointDirectionIn {
			inEp, err = intf.InEndpoint(ep.Number)
			if err != nil {
				intf.Close()
				cfg.Close()
				dev.Close()
				uctx.Close()
				return nil, fmt.Errorf("error getting IN endpoint %d: %w", ep.Number, err)
			}
		} else if ep.Direction == gousb.EndpointDirectionOut {
			outEp, err = intf.OutEndpoint(ep.Number)
			if err != nil {
				intf.Close()
				cfg.Close()
				dev.Close()
				uctx.Close()
				return nil, fmt.Errorf("error getting OUT endpoint %d: %w", ep.Number, err)
			}
		}
	}

	return &USB{
		ctx:    uctx,
		device: dev,
		cfg:    cfg,
		intf:   intf,
		inEp:   inEp,
		outEp:  outEp,
	}, nil
}

func (k *USB) Close() error {
	var errs error
	if k.intf != nil {
		k.intf.Close()
	}
	if k.cfg != nil {
		if err := k.cfg.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if k.device != nil {
		if err := k.device.Reset(); err != nil {
			errs = errors.Join(errs, err)
		}
		if err := k.device.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	if k.ctx != nil {
		if err := k.ctx.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (k *USB) write(ctx context.Context, data []byte) (int, error) {
	n, err := k.outEp.WriteContext(ctx, data)
	if err != nil {
		return n, fmt.Errorf("failed to write data: %w", err)
	}
	return n, nil
}

func (k *USB) read(ctx context.Context) ([]byte, error) {
	resp := make([]byte, k.inEp.Desc.MaxPacketSize)
	n, err := k.inEp.ReadContext(ctx, resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return resp[:n], nil
}
