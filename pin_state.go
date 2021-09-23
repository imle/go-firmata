package firmata

import (
	"fmt"
)

type PinStateResponse struct {
	Pin   uint8
	Mode  PinMode
	State int
}

func (p PinStateResponse) String() string {
	return fmt.Sprintf("pin(%d) mode(%s) state(%d)", p.Pin, p.Mode, p.State)
}
