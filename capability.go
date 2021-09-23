package firmata

import (
	"bytes"
	"fmt"
)

var pinModeOrder = []PinMode{
	PinModeDigitalInput,
	PinModeDigitalOutput,
	PinModeAnalogInput,
	PinModePWM,
	PinModeServo,
	PinModeShift,
	PinModeI2C,
	PinModeOneWire,
	PinModeStepper,
	PinModeEncoder,
	PinModeSerial,
	PinModeInputPullUp,
	PinModeSPI,
	PinModeSonar,
	PinModeTone,
	PinModeDHT,
}

const CapabilityResponsePinDelimiter = 0x7F

type CapabilityResponse struct {
	SupportedPinModes []map[PinMode]uint8
}

func (c CapabilityResponse) String() string {
	str := bytes.Buffer{}
	for pin, modeMap := range c.SupportedPinModes {
		_, _ = fmt.Fprintf(&str, "pin %2v: [", pin)
		if len(modeMap) > 0 {
			for _, mode := range pinModeOrder {
				if resolution, ok := modeMap[mode]; ok {
					_, _ = fmt.Fprintf(&str, "%s: %d, ", mode, resolution)
				}
			}
			str.Truncate(str.Len() - 2)
		}
		_, _ = fmt.Fprintf(&str, "]\n")
	}
	return str.String()
}
