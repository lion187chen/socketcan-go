package socketcan

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	maxDataLen = 8
	frameLen   = 16
)

type Frame struct {
	// ID is the CAN ID
	ID uint32
	// payload data.
	Data []byte
	// Whether a extended frame or not.
	IsExtended bool
	// Whether a remote frame or not.
	IsRemote bool
	// Whether a error frame or not.
	IsError bool
}

var pad [3]byte

// type frame struct {
// 	maskId uint32
// 	dlc    uint8
// 	// padding+reserved fields
// 	pad [3]byte
// 	// bytes contains the frame payload.
// 	data [maxDataLen]byte
// }

func (f *Frame) marshal() []byte {
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
	if len(f.Data) < maxDataLen {
		buf = append(buf, byte(len(f.Data)))
		buf = append(buf, pad[:]...)
	} else {
		buf = append(buf, byte(maxDataLen))
		buf = append(buf, pad[:]...)
	}

	td := make([]byte, maxDataLen)
	copy(td, f.Data)

	buf = append(buf, td...)
	return buf
}

func (f *Frame) unmarshal(bs []byte) *Frame {
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

	if dlc > maxDataLen {
		f.Data = make([]byte, maxDataLen)
	} else {
		f.Data = make([]byte, dlc)
	}
	copy(f.Data, bs[frameLen-maxDataLen:])
	return f
}
