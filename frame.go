package socketcan

import "encoding/binary"

const (
	MAX_DATA_LEN = 8
	frameLen     = 16

	CAN_EFF_FLAG uint32 = 0x80000000 // Extended frame flag.
	CAN_RTR_FLAG uint32 = 0x40000000 // Remote frame flag.
	CAN_ERR_FLAG uint32 = 0x20000000 // Error frame flag.
	/* mask */
	CAN_SFF_MASK uint32 = 0x000007FF // Use "can_id & CAN_SFF_MASK" to get standard frame ID.
	CAN_EFF_MASK uint32 = 0x1FFFFFFF // Use "can_id & CAN_EFF_MASK" to get extended frame ID.
	CAN_ERR_MASK uint32 = 0x1FFFFFFF // omit EFF, RTR, ERR flags.
)

type Frame struct {
	// ID is the CAN ID
	ID uint32
	// Length is the number of bytes of data in the frame.
	Length uint8
	// payload data.
	Data [MAX_DATA_LEN]byte
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
// 	data [MAX_DATA_LEN]byte
// }

func (f *Frame) marshal() []byte {
	var maskId uint32

	maskId = f.ID
	if f.IsRemote {
		maskId |= CAN_RTR_FLAG
	}
	if f.IsExtended {
		maskId |= CAN_EFF_FLAG
	}
	if f.IsError {
		maskId |= CAN_ERR_FLAG
	}

	var buf []byte
	buf = binary.LittleEndian.AppendUint32(buf, maskId)
	buf = append(buf, byte(f.Length))
	buf = append(buf, pad[:]...)
	buf = append(buf, f.Data[:]...)
	return buf
}
