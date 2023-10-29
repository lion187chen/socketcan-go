//go:build linux && go1.12

package canframe

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

const LINUX_FRAME_LEN = 16

var pad [3]byte

// type frame struct {
// 	maskId uint32
// 	dlc    uint8
// 	// padding+reserved fields
// 	pad [3]byte
// 	// bytes contains the frame payload.
// 	data [FRAME_MAX_DATA_LEN]byte
// }

func (f *Frame) Marshal() []byte {
	var maskId uint32

	maskId = f.ID
	if f.IsExtended {
		maskId |= unix.CAN_EFF_FLAG
	}
	if f.IsRemote {
		maskId |= unix.CAN_RTR_FLAG
	}
	if f.IsError {
		maskId |= unix.CAN_ERR_FLAG
	}

	var buf []byte

	buf = binary.LittleEndian.AppendUint32(buf, maskId)
	if len(f.Data) < FRAME_MAX_DATA_LEN {
		buf = append(buf, byte(len(f.Data)))
		buf = append(buf, pad[:]...)
	} else {
		buf = append(buf, byte(FRAME_MAX_DATA_LEN))
		buf = append(buf, pad[:]...)
	}

	td := make([]byte, FRAME_MAX_DATA_LEN)
	copy(td, f.Data)

	buf = append(buf, td...)
	return buf
}

func (f *Frame) Unmarshal(bs []byte) *Frame {
	f.ID = binary.LittleEndian.Uint32(bs[0:unsafe.Sizeof(f.ID)])

	if f.ID&unix.CAN_EFF_FLAG == unix.CAN_EFF_FLAG {
		f.IsExtended = true
	}
	if f.ID&unix.CAN_RTR_FLAG == unix.CAN_RTR_FLAG {
		f.IsRemote = true
	}
	if f.ID&unix.CAN_ERR_FLAG == unix.CAN_ERR_FLAG {
		f.IsError = true
	}

	f.ID = f.ID & unix.CAN_EFF_MASK
	f.ID = f.ID & unix.CAN_ERR_MASK

	dlc := bs[unsafe.Sizeof(f.ID)]

	if dlc > FRAME_MAX_DATA_LEN {
		f.Data = make([]byte, FRAME_MAX_DATA_LEN)
	} else {
		f.Data = make([]byte, dlc)
	}
	copy(f.Data, bs[LINUX_FRAME_LEN-FRAME_MAX_DATA_LEN:])
	return f
}
