package canframe

const FRAME_MAX_DATA_LEN = 8

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
