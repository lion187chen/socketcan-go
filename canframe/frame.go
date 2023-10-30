package canframe

const FRAME_MAX_DATA_LEN = 8

type Frame struct {
	// ID is the CAN ID
	ID uint32 `json:"id,omitempty"`
	// payload data.
	Data []byte `json:"data,omitempty"`
	// Whether a extended frame or not.
	IsExtended bool `json:"is_extended,omitempty"`
	// Whether a remote frame or not.
	IsRemote bool `json:"is_remote,omitempty"`
	// Whether a error frame or not.
	IsError bool `json:"is_error,omitempty"`
}
