package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/lion187chen/socketcan-go"
	"github.com/lion187chen/socketcan-go/canframe"
)

var lookback bool = true

var wg sync.WaitGroup

func main() {
	// Create a new CAN, the can interface name is "can0".
	can := new(socketcan.Can).Init("can0")
	// Set down to set bitrate.
	can.SetDown()
	// Test if CAN is down.
	fmt.Println(can.IsUp())
	// To set bitrate, we must down the CAN interface first.
	can.SetBitrate(100000)
	// Up the CAN to set filter or transmit CAN frames.
	can.SetUp()
	// To see CAN bitrate.
	fmt.Println(can.Bitrate())
	// Test if CAN is up.
	fmt.Println(can.IsUp())
	// Dial() will open a CAN socket and bind it to the given interface in Init().
	can.Dial()

	// After Dial(), we can set CAN hardware filters.
	var filters []socketcan.Filter = []socketcan.Filter{
		socketcan.NewExtFilter(0x20),
		socketcan.NewExtFilter(0x21),
		socketcan.NewExtFilter(0x40),
	}
	can.SetFilter(filters)

	// Or set the CAN in lookback mode.
	// We will stop lookback mode in EchoTsk.
	var sendFrame canframe.Frame = canframe.Frame{
		ID:         0x20,
		Data:       []byte{0x01, 0x02, 0x03, 0x4, 0x05, 0x06, 0x07, 0x08},
		IsExtended: true,
		IsRemote:   false,
		IsError:    false,
	}
	can.SetLoopback(lookback)
	// Send a frame. We will receive it immediately because we are in the lookback mode.
	n, err := can.SendFrame(&sendFrame)
	if err != nil {
		fmt.Println(n, err)
	}
	// Start a echo goroutine.
	wg.Add(1)
	go Echoroutine(can)

	// Wait 1 minute for CAN echo test.
	time.Sleep(1 * time.Minute)
	// Close the CAN.
	can.Close()
	// After the close operation, can.RcvFrame() will return an error, so we can exit from Echoroutine().
	// Wait until Echoroutine() done.
	wg.Wait()
}

func Echoroutine(can *socketcan.Can) {
	var err error = nil
	var frame canframe.Frame
	for err == nil {
		// can.RcvFrame() will block until new datas arrived.
		frame, err = can.RcvFrame()
		if lookback {
			lookback = false
			can.SetLoopback(lookback)
		}

		fmt.Println(frame)
		n, err := can.SendFrame(&frame)
		if err != nil {
			fmt.Println(n, err)
		}
	}
	fmt.Println("Echoroutine Exit:", err)
	wg.Done()
}
