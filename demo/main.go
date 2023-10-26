package main

import "github.com/lion187chen/socketcan-go"

func main() {
	sc := new(socketcan.Device).Init("can0")
	sc.SetDown()
	sc.SetUp()
	sc.Dial()
	var sendFrame socketcan.Frame = socketcan.Frame{
		ID:         0x20,
		Length:     4,
		Data:       [socketcan.MAX_DATA_LEN]byte{0x01, 0x02, 0x03, 0x55},
		IsExtended: true,
		IsRemote:   false,
		IsError:    false,
	}
	sc.SendFrame(&sendFrame)
}
