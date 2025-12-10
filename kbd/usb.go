/*
Copyright 2025 Jochen Eisinger

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.
3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software without
specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS  AS IS  AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Package kbd implements interaction with the keyboard. It is a 1-to-1 rewrite
// of the python code in zuoya_gmk87.py.
package kbd

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/gousb"
)

type usbDevice struct {
	ctx    *gousb.Context
	device *gousb.Device
	cfg    *gousb.Config
	intf   *gousb.Interface
	inEp   *gousb.InEndpoint
	outEp  *gousb.OutEndpoint
}

func newUSBDevice(vid, pid gousb.ID) (*usbDevice, error) {
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

	return &usbDevice{
		ctx:    uctx,
		device: dev,
		cfg:    cfg,
		intf:   intf,
		inEp:   inEp,
		outEp:  outEp,
	}, nil
}

func (k *usbDevice) Close() error {
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

func (k *usbDevice) write(ctx context.Context, data []byte) (int, error) {
	n, err := k.outEp.WriteContext(ctx, data)
	if err != nil {
		return n, fmt.Errorf("failed to write data: %w", err)
	}
	return n, nil
}

func (k *usbDevice) read(ctx context.Context) ([]byte, error) {
	resp := make([]byte, k.inEp.Desc.MaxPacketSize)
	n, err := k.inEp.ReadContext(ctx, resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return resp[:n], nil
}
