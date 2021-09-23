package firmata

type PinMode uint16

const (
	PinModeDigitalInput  PinMode = 0x0
	PinModeDigitalOutput PinMode = 0x1
	PinModeAnalogInput   PinMode = 0x2
	PinModePWM           PinMode = 0x3
	PinModeServo         PinMode = 0x4
	PinModeShift         PinMode = 0x5
	PinModeI2C           PinMode = 0x6
	PinModeOneWire       PinMode = 0x7
	PinModeStepper       PinMode = 0x8
	PinModeEncoder       PinMode = 0x9
	PinModeSerial        PinMode = 0xA
	PinModeInputPullUp   PinMode = 0xB
	PinModeSPI           PinMode = 0xC
	PinModeSonar         PinMode = 0xD
	PinModeTone          PinMode = 0xE
	PinModeDHT           PinMode = 0xF
)

var pinModeToStringMap = map[PinMode]string{
	PinModeDigitalInput:  "DigitalInput",
	PinModeDigitalOutput: "DigitalOutput",
	PinModeAnalogInput:   "AnalogInput",
	PinModePWM:           "PWM",
	PinModeServo:         "Servo",
	PinModeShift:         "Shift",
	PinModeI2C:           "I2C",
	PinModeOneWire:       "OneWire",
	PinModeStepper:       "Stepper",
	PinModeEncoder:       "Encoder",
	PinModeSerial:        "Serial",
	PinModeInputPullUp:   "InputPullUp",
	PinModeSPI:           "SPI",
	PinModeSonar:         "Sonar",
	PinModeTone:          "Tone",
	PinModeDHT:           "DHT",
}

func (p PinMode) String() string {
	if v, ok := pinModeToStringMap[p]; ok {
		return v
	}
	return "Unknown"
}
