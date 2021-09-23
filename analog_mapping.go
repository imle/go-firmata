package firmata

import (
	"bytes"
	"fmt"
)

type AnalogMappingResponse struct {
	PinMapping []uint8
}

func (a AnalogMappingResponse) String() string {
	str := bytes.Buffer{}
	for analogPin, digitalPin := range a.PinMapping {
		_, _ = fmt.Fprintf(&str, "A%d: %d\n", analogPin, digitalPin)
	}
	return str.String()
}

type ExtendedAnalogMappingResponse struct {
	Pin uint8
}

func (a ExtendedAnalogMappingResponse) String() string {
	return fmt.Sprintf("%d", a.Pin)
}
