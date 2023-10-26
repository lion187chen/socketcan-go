//go:build linux && go1.12

package socketcan

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

// Device
type Device struct {
	fd    int
	nface *net.Interface
}

// Device public.

func (d *Device) Init(device string) *Device {
	var err error
	d.nface, err = net.InterfaceByName(device)
	if err != nil {
		return nil
	}
	return d
}

func (d *Device) SetUp() error {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	ifi := &ifInfoMsg{
		Index:  int32(d.nface.Index),
		Flags:  unix.IFF_UP,
		Change: unix.IFF_UP,
	}
	req, err := d.newRequest(unix.RTM_NEWLINK, ifi)
	if err != nil {
		return fmt.Errorf("couldn't create netlink request: %w", err)
	}

	res, err := c.Execute(req)
	if err != nil {
		return fmt.Errorf("couldn't set link up: %w", err)
	}
	if len(res) > 1 {
		return fmt.Errorf("expected 1 message, got %d", len(res))
	}
	return nil
}

func (d *Device) SetDown() error {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	ifi := &ifInfoMsg{
		Index:  int32(d.nface.Index),
		Flags:  0,
		Change: unix.IFF_UP,
	}
	req, err := d.newRequest(unix.RTM_NEWLINK, ifi)
	if err != nil {
		return fmt.Errorf("couldn't create netlink request: %w", err)
	}

	res, err := c.Execute(req)
	if err != nil {
		return fmt.Errorf("couldn't set link down: %w", err)
	}
	if len(res) > 1 {
		return fmt.Errorf("expected 1 message, got %d", len(res))
	}
	return nil
}

func (d *Device) Dial() (err error) {
	d.fd, err = unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	err = unix.Bind(d.fd, &unix.SockaddrCAN{Ifindex: d.nface.Index})
	if err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}

func (d *Device) SendFrame(f *Frame) {
	unix.Write(d.fd, f.marshal())
}

// Device private.

func (d *Device) newRequest(typ netlink.HeaderType, ifi *ifInfoMsg) (netlink.Message, error) {
	req := netlink.Message{
		Header: netlink.Header{
			Flags: netlink.Request | netlink.Acknowledge,
			Type:  typ,
		},
	}
	msg := ifi.marshalBinary()
	req.Data = append(req.Data, msg...)
	return req, nil
}

// ifInfoMsg
type ifInfoMsg unix.IfInfomsg

func (ifi *ifInfoMsg) marshalBinary() []byte {
	buf := make([]byte, 2)
	buf[0] = ifi.Family
	buf[1] = 0 // reserved
	buf = binary.LittleEndian.AppendUint16(buf, ifi.Type)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ifi.Index))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ifi.Flags))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ifi.Change))
	return buf
}
