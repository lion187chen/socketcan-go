//go:build linux && go1.12

package socketcan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
	"unsafe"

	"github.com/lion187chen/socketcan-go/canframe"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

// Can
type Can struct {
	fd    int
	nface *net.Interface
}

// Can public.

const (
	canLinkType  = "can"
	vcanLinkType = "vcan"
)

// ifName is the CAN interface name, such as "can0", "can1"...
func (my *Can) Init(ifName string) *Can {
	var err error
	my.nface, err = net.InterfaceByName(ifName)
	if err != nil {
		return nil
	}
	return my
}

// Up the CAN interface.
func (my *Can) SetUp() error {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	ifi := &ifInfoMsg{
		Index:  int32(my.nface.Index),
		Flags:  unix.IFF_UP,
		Change: unix.IFF_UP,
	}
	req, err := my.newRequest(unix.RTM_NEWLINK, ifi)
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

// Down th CAN interface.
func (my *Can) SetDown() error {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	ifi := &ifInfoMsg{
		Index:  int32(my.nface.Index),
		Flags:  0,
		Change: unix.IFF_UP,
	}
	req, err := my.newRequest(unix.RTM_NEWLINK, ifi)
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

func (my *Can) IsUp() (bool, error) {
	_, ifInfo, err := my.updateInfo()
	if err != nil {
		return false, err
	}
	if ifInfo.Flags&unix.IFF_UP != 0 {
		return true, nil
	}
	return false, nil
}

// Get CAN interface's info.
func (my *Can) Info() (Info, error) {
	info, _, err := my.updateInfo()
	if err != nil {
		return Info{}, err
	}
	return *info, nil
}

// Get current bitrate.
func (my *Can) Bitrate() (uint32, error) {
	lkInf, _, err := my.updateInfo()
	if err != nil {
		return 0, fmt.Errorf("couldn't retrieve bitrate: %w", err)
	}
	return lkInf.BitTiming.Bitrate, nil
}

// To set bitrate, you must down the CAN interface first.
func (my *Can) SetBitrate(bitrate uint32) error {
	ifi := &ifInfoMsg{
		Index: int32(my.nface.Index),
	}
	req, err := my.newRequest(unix.RTM_NEWLINK, ifi)
	if err != nil {
		return fmt.Errorf("couldn't create netlink request: %w", err)
	}

	info, err := my.initSetParameters()
	if err != nil {
		return fmt.Errorf("couldn't get current parameters: %w", err)
	}

	info.BitTiming.Bitrate = bitrate
	ae := netlink.NewAttributeEncoder()
	ae.Nested(unix.IFLA_LINKINFO, info.encode)
	liData, err := ae.Encode()
	if err != nil {
		return fmt.Errorf("couldn't encode message: %w", err)
	}
	req.Data = append(req.Data, liData...)

	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	res, err := c.Execute(req)
	if err != nil {
		return fmt.Errorf("couldn't set bitrate: %w", err)
	}
	if len(res) > 1 {
		return fmt.Errorf("expected 1 message, got %d", len(res))
	}
	return nil
}

// Dial() will open a CAN socket and bind it to the given interface in Init().
func (my *Can) Dial() (err error) {
	my.fd, err = unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		return fmt.Errorf("socket: %w", err)
	}

	err = unix.Bind(my.fd, &unix.SockaddrCAN{Ifindex: my.nface.Index})
	if err != nil {
		return fmt.Errorf("bind: %w", err)
	}

	return nil
}

// True for lookback mode.
func (my *Can) SetLoopback(enable bool) error {
	value := 0
	if enable {
		value = 1
	}
	err := unix.SetsockoptInt(my.fd, unix.SOL_CAN_RAW, unix.CAN_RAW_RECV_OWN_MSGS, value)
	return err
}

// You can use NewStdFilter, NewStdInvFilter(), NewExtFilter(), NewExtInvFilter() help functions to create []Filter.
func (my *Can) SetFilter(fs []Filter) error {
	return setsockopt(my.fd, unix.SOL_CAN_RAW, unix.CAN_RAW_FILTER, unsafe.Pointer(&fs[0]), uintptr(len(fs))*unsafe.Sizeof(Filter{}))
}

// Set CAN receive timeout.
func (my *Can) SetRecvTimeout(timeout time.Duration) error {
	tv := unix.NsecToTimeval(timeout.Nanoseconds())
	err := unix.SetsockoptTimeval(my.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv)
	return err
}

// Set CAN send timeout.
func (my *Can) SetSendTimeout(timeout time.Duration) error {
	tv := unix.NsecToTimeval(timeout.Nanoseconds())
	err := unix.SetsockoptTimeval(my.fd, unix.SOL_SOCKET, unix.SO_SNDTIMEO, &tv)
	return err
}

// Set the CAN in listen only mode.
func (my *Can) SetListenOnlyMode(mode bool) error {

	ifi := &ifInfoMsg{
		Index: int32(my.nface.Index),
	}
	req, err := my.newRequest(unix.RTM_NEWLINK, ifi)
	if err != nil {
		return fmt.Errorf("couldn't create netlink request: %w", err)
	}

	info, err := my.initSetParameters()
	if err != nil {
		return fmt.Errorf("couldn't get current parameters: %w", err)
	}

	if mode {
		info.CtrlMode.Mask |= unix.CAN_CTRLMODE_LISTENONLY
		info.CtrlMode.Flags |= unix.CAN_CTRLMODE_LISTENONLY
	} else {
		info.CtrlMode.Mask |= unix.CAN_CTRLMODE_LISTENONLY
		info.CtrlMode.Flags = 0
	}

	ae := netlink.NewAttributeEncoder()
	ae.Nested(unix.IFLA_LINKINFO, info.encode)
	liData, err := ae.Encode()
	if err != nil {
		return fmt.Errorf("couldn't encode message: %w", err)
	}

	req.Data = append(req.Data, liData...)

	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	res, err := c.Execute(req)
	if err != nil {
		return fmt.Errorf("couldn't set listen-only mode: %w", err)
	}
	if len(res) > 1 {
		return fmt.Errorf("expected 1 message, got %d", len(res))
	}
	return nil
}

// SendFrame will block until write done or a error occured.
func (my *Can) SendFrame(f *canframe.Frame) {
	unix.Write(my.fd, f.Marshal())
}

// RcvFrame() will block until new datas arrived or a error occured.
func (my *Can) RcvFrame() (canframe.Frame, error) {
	rd := make([]byte, canframe.LINUX_FRAME_LEN)
	n, err := unix.Read(my.fd, rd)

	var f canframe.Frame
	if err != nil {
		return f, err
	}
	if n != 16 {
		return f, errors.New("read not done")
	}

	return *(f.Unmarshal(rd)), nil
}

// After all, we must close the CAN.
func (my *Can) Close() error {
	return unix.Close(my.fd)
}

// Can private.

func (my *Can) newRequest(typ netlink.HeaderType, ifi *ifInfoMsg) (netlink.Message, error) {
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

func (my *Can) updateInfo() (*Info, *ifInfoMsg, error) {
	c, err := netlink.Dial(unix.NETLINK_ROUTE, &netlink.Config{})
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't dial netlink socket: %w", err)
	}
	defer c.Close()

	ifi := &ifInfoMsg{
		Index: int32(my.nface.Index),
	}
	req, err := my.newRequest(unix.RTM_GETLINK, ifi)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create netlink request: %w", err)
	}

	res, err := c.Execute(req)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't retrieve link info: %w", err)
	}
	if len(res) > 1 {
		return nil, nil, fmt.Errorf("expected 1 message, got %d", len(res))
	}

	info, ifInfo, err := respondData(res[0].Data).unmarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't decode info: %w", err)
	}
	return info, ifInfo, nil
}

func (my *Can) initSetParameters() (Info, error) {
	info, err := my.Info()
	if err != nil {
		return Info{}, err
	}

	return Info{
		BitTiming: CanBitTiming{
			Bitrate: info.BitTiming.Bitrate,
		},
		CtrlMode: info.CtrlMode,
	}, nil
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

// Info

type Info struct {
	DevName        string
	BitTiming      CanBitTiming
	BitTimingConst CanBitTimingConst
	Clock          CanClock
	CtrlMode       CanCtrlMode
	ErrCounters    CanBusErrCounters
	DevStats       CanDevStats
}

func (li *Info) encode(nae *netlink.AttributeEncoder) error {
	nae.String(unix.IFLA_INFO_KIND, canLinkType)
	nae.Nested(unix.IFLA_INFO_DATA, li.encodeData)
	return nil
}

func (li *Info) decode(nad *netlink.AttributeDecoder) error {
	var err error
	for nad.Next() {
		switch nad.Type() {
		case unix.IFLA_INFO_KIND:
			iType := nad.String()
			if (iType != canLinkType) && (iType != vcanLinkType) {
				return fmt.Errorf("not a CAN interface")
			}
		case unix.IFLA_INFO_DATA:
			nad.Nested(li.decodeData)
		case unix.IFLA_INFO_XSTATS:
			err = li.DevStats.unmarshalBinary(nad.Bytes())
		default:
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// respondData

type respondData []byte

func (rd respondData) unmarshalBinary() (*Info, *ifInfoMsg, error) {
	var info Info
	var ifInfo ifInfoMsg
	if err := ifInfo.unmarshalBinary(rd[:unix.SizeofIfInfomsg]); err != nil {
		return nil, nil, fmt.Errorf("couldn't unmarshal ifInfoMsg: %w", err)
	}

	ad, err := netlink.NewAttributeDecoder(rd[unix.SizeofIfInfomsg:])
	if err != nil {
		return nil, nil, err
	}
	if ifInfo.Type != unix.ARPHRD_CAN {
		return nil, nil, fmt.Errorf("not a CAN interface")
	}
	for ad.Next() {
		switch ad.Type() {
		case unix.IFLA_IFNAME:
			info.DevName = ad.String()
		case unix.IFLA_LINKINFO:
			ad.Nested(info.decode)
		default:
		}
	}
	if err := ad.Err(); err != nil {
		return nil, nil, fmt.Errorf("couldn't decode link: %w", err)
	}
	return &info, &ifInfo, nil
}

func (i *Info) encodeData(nae *netlink.AttributeEncoder) error {
	nae.Bytes(unix.IFLA_CAN_BITTIMING, i.BitTiming.marshalBinary())
	nae.Bytes(unix.IFLA_CAN_CTRLMODE, i.CtrlMode.marshalBinary())
	return nil
}

func (i *Info) decodeData(nad *netlink.AttributeDecoder) error {
	var err error
	for nad.Next() {
		switch nad.Type() {
		case unix.IFLA_CAN_BITTIMING:
			err = i.BitTiming.unmarshalBinary(nad.Bytes())
		case unix.IFLA_CAN_BITTIMING_CONST:
			err = i.BitTimingConst.unmarshalBinary(nad.Bytes())
		case unix.IFLA_CAN_CLOCK:
			err = i.Clock.unmarshalBinary(nad.Bytes())
		case unix.IFLA_CAN_CTRLMODE:
			err = i.CtrlMode.unmarshalBinary(nad.Bytes())
		case unix.IFLA_CAN_BERR_COUNTER:
			err = i.ErrCounters.unmarshalBinary(nad.Bytes())
		default:
		}
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	sizeOfBitTiming        = int(unsafe.Sizeof(CanBitTiming{}))
	sizeOfBitTimingConst   = int(unsafe.Sizeof(CanBitTimingConst{}))
	sizeOfClock            = int(unsafe.Sizeof(CanClock{}))
	sizeOfCtrlMode         = int(unsafe.Sizeof(CanCtrlMode{}))
	sizeOfBusErrorCounters = int(unsafe.Sizeof(CanBusErrCounters{}))
	sizeOfStats            = int(unsafe.Sizeof(CanDevStats{}))
)

// CanBitTiming

type CanBitTiming unix.CANBitTiming

func (bt *CanBitTiming) marshalBinary() []byte {
	buf := make([]byte, sizeOfBitTiming)
	nlenc.PutUint32(buf[0:4], bt.Bitrate)
	nlenc.PutUint32(buf[4:8], bt.Sample_point)
	nlenc.PutUint32(buf[8:12], bt.Tq)
	nlenc.PutUint32(buf[12:16], bt.Prop_seg)
	nlenc.PutUint32(buf[16:20], bt.Phase_seg1)
	nlenc.PutUint32(buf[20:24], bt.Phase_seg2)
	nlenc.PutUint32(buf[24:28], bt.Sjw)
	nlenc.PutUint32(buf[28:32], bt.Brp)
	return buf
}

func (bt *CanBitTiming) unmarshalBinary(data []byte) error {
	if len(data) != sizeOfBitTiming {
		return fmt.Errorf(
			"data is not a valid CanBitTiming, expected: %d bytes, got: %d bytes",
			sizeOfBitTiming,
			len(data),
		)
	}
	bt.Bitrate = nlenc.Uint32(data[0:4])
	bt.Sample_point = nlenc.Uint32(data[4:8])
	bt.Tq = nlenc.Uint32(data[8:12])
	bt.Prop_seg = nlenc.Uint32(data[12:16])
	bt.Phase_seg1 = nlenc.Uint32(data[16:20])
	bt.Phase_seg2 = nlenc.Uint32(data[20:24])
	bt.Sjw = nlenc.Uint32(data[24:28])
	bt.Brp = nlenc.Uint32(data[28:32])
	return nil
}

// CanBitTimingConst

type CanBitTimingConst unix.CANBitTimingConst

func (btc *CanBitTimingConst) unmarshalBinary(data []byte) error {
	if len(data) != sizeOfBitTimingConst {
		return fmt.Errorf(
			"data is not a valid CanBitTimingConst, expected: %d bytes, got: %d bytes",
			sizeOfBitTimingConst,
			len(data),
		)
	}
	copy(btc.Name[:], data[0:16])
	btc.Tseg1_min = nlenc.Uint32(data[16:20])
	btc.Tseg1_max = nlenc.Uint32(data[20:24])
	btc.Tseg2_min = nlenc.Uint32(data[24:28])
	btc.Tseg2_max = nlenc.Uint32(data[28:32])
	btc.Sjw_max = nlenc.Uint32(data[32:36])
	btc.Brp_min = nlenc.Uint32(data[36:40])
	btc.Brp_max = nlenc.Uint32(data[40:44])
	btc.Brp_inc = nlenc.Uint32(data[44:48])
	return nil
}

// CanClock

type CanClock unix.CANClock

func (c *CanClock) unmarshalBinary(data []byte) error {
	if len(data) != sizeOfClock {
		return fmt.Errorf(
			"data is not a valid CanClock, expected: %d bytes, got: %d bytes",
			sizeOfClock,
			len(data),
		)
	}
	c.Freq = nlenc.Uint32(data)
	return nil
}

// CanBusErrCounters

type CanBusErrCounters unix.CANBusErrorCounters

func (bec *CanBusErrCounters) unmarshalBinary(data []byte) error {
	if len(data) != sizeOfBusErrorCounters {
		return fmt.Errorf(
			"data is not a valid CanBusErrCounters, expected: %d bytes, got: %d bytes",
			sizeOfBusErrorCounters,
			len(data),
		)
	}
	bec.Txerr = nlenc.Uint16(data[0:2])
	bec.Rxerr = nlenc.Uint16(data[2:4])
	return nil
}

// CanCtrlMode

type CanCtrlMode unix.CANCtrlMode

func (cm *CanCtrlMode) marshalBinary() []byte {
	buf := make([]byte, sizeOfCtrlMode)
	nlenc.PutUint32(buf[0:4], cm.Mask)
	nlenc.PutUint32(buf[4:8], cm.Flags)
	return buf
}

func (cm *CanCtrlMode) unmarshalBinary(data []byte) error {
	if len(data) != sizeOfCtrlMode {
		return fmt.Errorf(
			"data is not a valid CanCtrlMode, expected: %d bytes, got: %d bytes",
			sizeOfCtrlMode,
			len(data),
		)
	}
	cm.Mask = nlenc.Uint32(data[0:4])
	cm.Flags = nlenc.Uint32(data[4:8])
	return nil
}

// CanDevStats

type CanDevStats unix.CANDeviceStats

func (s *CanDevStats) unmarshalBinary(data []byte) error {
	if len(data) != sizeOfStats {
		return fmt.Errorf(
			"data is not a valid CanDevStats, expected: %d bytes, got: %d bytes",
			sizeOfStats,
			len(data),
		)
	}
	s.Bus_error = nlenc.Uint32(data[0:4])
	s.Error_warning = nlenc.Uint32(data[4:8])
	s.Error_passive = nlenc.Uint32(data[8:12])
	s.Bus_off = nlenc.Uint32(data[12:16])
	s.Arbitration_lost = nlenc.Uint32(data[16:20])
	s.Restarts = nlenc.Uint32(data[20:24])
	return nil
}

func (ifi *ifInfoMsg) unmarshalBinary(data []byte) error {
	if len(data) != unix.SizeofIfInfomsg {
		return fmt.Errorf(
			"data is not a valid ifInfoMsg, expected: %d bytes, got: %d bytes",
			unix.SizeofIfInfomsg,
			len(data),
		)
	}
	ifi.Family = nlenc.Uint8(data[0:1])
	ifi.Type = nlenc.Uint16(data[2:4])
	ifi.Index = nlenc.Int32(data[4:8])
	ifi.Flags = nlenc.Uint32(data[8:12])
	ifi.Change = nlenc.Uint32(data[12:16])
	return nil
}
