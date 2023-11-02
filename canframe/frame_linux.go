//go:build linux && go1.12

package canframe

import (
	"bytes"
	"encoding/binary"

	"golang.org/x/sys/unix"
)

const LINUX_FRAME_LEN = 16

type rawFrame struct {
	MaskId uint32
	Dlc    uint8
	// padding+reserved fields
	Pad [3]byte
	// bytes contains the frame payload.
	Data [FRAME_MAX_DATA_LEN]byte
}

func (f *Frame) Marshal() ([]byte, error) {
	var frame rawFrame

	frame.MaskId = f.ID
	if f.IsExtended {
		frame.MaskId |= unix.CAN_EFF_FLAG
	}
	if f.IsRemote {
		frame.MaskId |= unix.CAN_RTR_FLAG
	}
	if f.IsError {
		frame.MaskId |= unix.CAN_ERR_FLAG
	}

	if len(f.Data) < FRAME_MAX_DATA_LEN {
		frame.Dlc = uint8(len(f.Data))
	} else {
		frame.Dlc = uint8(FRAME_MAX_DATA_LEN)
	}
	copy(frame.Data[:], f.Data)

	var frameb bytes.Buffer
	err := binary.Write(&frameb, binary.LittleEndian, &frame)
	if err != nil {
		return nil, err
	}
	return frameb.Bytes(), nil
}

func (f *Frame) Unmarshal(bs []byte) error {
	var frame rawFrame
	var frameb bytes.Buffer

	frameb.Write(bs)
	err := binary.Read(&frameb, binary.LittleEndian, &frame)
	if err != nil {
		return err
	}

	f.ID = frame.MaskId
	f.ID &= ^(uint32(unix.CAN_EFF_FLAG | unix.CAN_RTR_FLAG | unix.CAN_ERR_FLAG))

	if frame.MaskId&unix.CAN_EFF_FLAG == unix.CAN_EFF_FLAG {
		f.IsExtended = true
	}
	if frame.MaskId&unix.CAN_RTR_FLAG == unix.CAN_RTR_FLAG {
		f.IsRemote = true
	}
	if frame.MaskId&unix.CAN_ERR_FLAG == unix.CAN_ERR_FLAG {
		f.IsError = true
	}

	if frame.Dlc < FRAME_MAX_DATA_LEN {
		f.Data = make([]byte, frame.Dlc)
	} else {
		f.Data = make([]byte, FRAME_MAX_DATA_LEN)
	}
	copy(f.Data, frame.Data[:])
	return err
}
