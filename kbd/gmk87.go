package kbd

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"runtime/trace"
	"time"
)

const (
	vid        = 0x320f
	pid        = 0x5055
	kbIfaceNum = 3
)

const (
	dateOffset = 35
)

type GMK87 struct {
	usb *USB
}

func NewKeyboard() (*GMK87, error) {
	usb, err := NewUSB(vid, pid)
	if err != nil {
		return nil, err
	}
	return &GMK87{usb: usb}, nil
}

func (k *GMK87) Close() error {
	return k.usb.Close()
}

func (k *GMK87) sendCommand(ctx context.Context, id byte, data []byte, pos uint32) ([]byte, error) {
	const (
		hdr    = 8
		maxLen = 64 - hdr
	)
	if len(data) > maxLen {
		return nil, fmt.Errorf("data too long: %d bytes", len(data))
	}

	if id == cmdEnd {
		time.Sleep(100 * time.Millisecond)
	}

	buf := make([]byte, 64)
	buf[0] = 0x04
	buf[3] = id
	buf[4] = byte(len(data))
	buf[5] = byte(pos & 0xff)
	buf[6] = byte((pos >> 8) & 0xff)
	buf[7] = byte((pos >> 16) & 0xff)
	copy(buf[8:], data)

	cksum := uint16(0)
	for i := 3; i < len(buf)-1; i++ {
		cksum += uint16(buf[i])
	}
	buf[1] = byte(cksum & 0xff)
	buf[2] = byte((cksum >> 8) & 0xff)

	n, err := k.usb.write(ctx, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write command %d: %w", id, err)
	}
	if n != len(buf) {
		return nil, fmt.Errorf("short write for command %d: wrote %d bytes, expected %d", id, n, len(buf))
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	for {
		resp, err := k.usb.read(ctx)
		slog.Debug("read response", "command", id, "bytes", len(resp), "err", err)
		if err != nil {
			return nil, fmt.Errorf("failed to read response for command %d: %w", id, err)
		}
		if len(resp) < 4 {
			return nil, fmt.Errorf("short read for command %d: read %d bytes", id, len(resp))
		}
		if !bytes.Equal(resp[0:3], buf[0:3]) {
			slog.Debug("ignoring unmatched response", "expected", buf[0:3], "got", resp[0:3])
			continue
		}
		slog.Debug("matched response", "command", id, "data", fmt.Sprintf("%x", resp[4:]))
		return resp[4:], nil
	}
}

const (
	cmdStart       = 1
	cmdEnd         = 2
	cmdUnknown3    = 3
	cmdConfigRead  = 5
	cmdConfigWrite = 6
)

func (k *GMK87) LoadConfig(ctx context.Context) ([]byte, error) {
	if _, err := k.sendCommand(ctx, cmdStart, nil, 0); err != nil {
		return nil, fmt.Errorf("failed to send load command: %w", err)
	}

	var fourBytes [4]byte

	for i := range 9 {
		if _, err := k.sendCommand(ctx, cmdUnknown3, fourBytes[:], uint32(i*4)); err != nil {
			return nil, fmt.Errorf("failed to send command 3 at position %d: %w", i*4, err)
		}
	}

	if _, err := k.sendCommand(ctx, cmdUnknown3, []byte{0x00}, 36); err != nil {
		return nil, fmt.Errorf("failed to send command 3 at position 36: %w", err)
	}

	if _, err := k.sendCommand(ctx, cmdEnd, nil, 0); err != nil {
		return nil, fmt.Errorf("failed to send finalize command: %w", err)
	}

	buffer := make([]byte, 0)
	for i := 0; i < 12; i++ {
		resp, err := k.sendCommand(ctx, cmdConfigRead, fourBytes[:], uint32(i*4))
		if err != nil {
			return nil, fmt.Errorf("failed to send command 5 at position %d: %w", i*4, err)
		}
		buffer = append(buffer, resp...)
	}

	return buffer, nil
}

func (k *GMK87) updateConfig(ctx context.Context, cfg []byte) error {
	if _, err := k.sendCommand(ctx, cmdStart, nil, 0); err != nil {
		return fmt.Errorf("failed to send update command: %w", err)
	}
	if _, err := k.sendCommand(ctx, cmdConfigWrite, cfg, 0); err != nil {
		return fmt.Errorf("failed to send config write command: %w", err)
	}
	if _, err := k.sendCommand(ctx, cmdEnd, nil, 0); err != nil {
		return fmt.Errorf("failed to send finalize command: %w", err)
	}
	return nil
}

func (k *GMK87) SetTime(ctx context.Context, t time.Time) error {
	ctx, task := trace.NewTask(ctx, "SetTime")
	defer task.End()

	opstart := time.Now()

	rgnLoad := trace.StartRegion(ctx, "loadConfig")
	cfg, err := k.LoadConfig(ctx)
	rgnLoad.End()
	opdur := time.Since(opstart)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// adjust time for operation duration
	t = t.Add(opdur * 2) // 2 operations read/write with ~opdur each

	cfg[dateOffset] = bcdEncode(t.Second())
	cfg[dateOffset+1] = bcdEncode(t.Minute())
	cfg[dateOffset+2] = bcdEncode(t.Hour())
	weekday := t.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	cfg[dateOffset+3] = byte(weekday)
	cfg[dateOffset+4] = bcdEncode(t.Day())
	cfg[dateOffset+5] = bcdEncode(int(t.Month()))
	cfg[dateOffset+6] = bcdEncode(t.Year() - 2000)

	rgnUpdate := trace.StartRegion(ctx, "updateConfig")
	if err := k.updateConfig(ctx, cfg); err != nil {
		rgnUpdate.End()
		return fmt.Errorf("failed to update config: %w", err)
	}
	rgnUpdate.End()

	return nil
}

func bcdEncode(v int) byte {
	if v < 0 || v > 99 {
		panic("bcdEncode: value out of range")
	}
	return byte((v/10)<<4 | (v % 10))
}
