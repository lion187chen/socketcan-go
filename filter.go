package socketcan

import "golang.org/x/sys/unix"

type Filter struct {
	Id   uint32
	Mask uint32
}

func NewStdFilter(id uint32) Filter {
	return Filter{
		Id:   id,
		Mask: unix.CAN_SFF_MASK,
	}
}

func NewStdInvFilter(id uint32) Filter {
	return Filter{
		Id:   id | unix.CAN_INV_FILTER,
		Mask: unix.CAN_SFF_MASK,
	}
}

func NewExtFilter(id uint32) Filter {
	return Filter{
		Id:   id,
		Mask: unix.CAN_EFF_MASK,
	}
}

func NewExtInvFilter(id uint32) Filter {
	return Filter{
		Id:   id | unix.CAN_INV_FILTER,
		Mask: unix.CAN_EFF_MASK,
	}
}
