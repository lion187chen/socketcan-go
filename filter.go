package socketcan

import "golang.org/x/sys/unix"

type Filter struct {
	Id   uint32
	Mask uint32
}

func NewStdFilter(id uint8) Filter {
	return Filter{
		Id:   uint32(id),
		Mask: unix.CAN_SFF_MASK,
	}
}

func NewStdInvFilter(id uint8) Filter {
	return Filter{
		Id:   uint32(id) | unix.CAN_INV_FILTER,
		Mask: unix.CAN_SFF_MASK,
	}
}

func NewExtFilter(id uint16) Filter {
	return Filter{
		Id:   uint32(id),
		Mask: unix.CAN_EFF_MASK,
	}
}

func NewExtInvFilter(id uint8) Filter {
	return Filter{
		Id:   uint32(id) | unix.CAN_INV_FILTER,
		Mask: unix.CAN_EFF_MASK,
	}
}
