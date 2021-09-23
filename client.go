package firmata

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	ErrDeviceDisconnected                 = errors.New("device disconnected")
	ErrUnsupportedFeature                 = errors.New("unsupported feature")
	ErrInvalidMessageTypeStart            = errors.New("invalid message type start")
	ErrNoDataRead                         = errors.New("no data read")
	ErrUnexpectedSysExMessageTypeReceived = errors.New("unexpected sysex message type")
	ErrAlreadyStarted                     = errors.New("client already started")
	ErrValueOutOfRange                    = errors.New("value is out of range")
	ErrNoI2CListenerForAddress            = errors.New("no i2c listener registered for address")
)

const (
	MaxUInt8  uint8  = (1<<8 - 1) >> 1
	MaxUInt16 uint16 = (1<<16 - 1) >> 2
)

var commandResponseMap = map[SysExCmd]SysExCmd{
	SysExAnalogMappingQuery: SysExAnalogMappingResponse,
	SysExCapabilityQuery:    SysExCapabilityResponse,
	SysExPinStateQuery:      SysExPinStateResponse,
}

type ClientI interface {
	SendSysEx(SysExCmd, ...byte) (chan []byte, error)
	SendReset() error
	ExtendedReportAnalogPin(uint8, int) error
	CapabilityQuery() (chan CapabilityResponse, error)
	PinStateQuery(uint8) (chan PinStateResponse, error)
	ReportFirmware() (chan FirmwareReport, error)
	SetPinMode(uint8, PinMode) error
	SetAnalogPinReporting(uint8, bool) error
	SetDigitalPinReporting(uint8, bool) error
	SetDigitalPortReporting(uint8, bool) error
	SetSamplingInterval(uint16) error
}

type Client struct {
	board                 io.ReadWriteCloser
	responseChannels      map[SysExCmd][]chan []byte
	sysExListenerChannels map[SysExCmd]chan []byte
	i2cListeners          map[uint8]chan []byte

	mu      sync.Mutex
	started bool

	// We want to report these to the requester, but also save them for internal use.
	cr  CapabilityResponse
	amr AnalogMappingResponse
}

func (c *Client) SetPinMode(pin uint8, mode PinMode) error {
	return c.write([]uint8{uint8(SetPinMode), pin, uint8(mode)}, nil)
}

func (c *Client) SetDigitalPinValue(pin uint8, value bool) error {
	v := byte(0)
	if value {
		v = 1
	}
	return c.write([]uint8{uint8(SetDigitalPinValue), pin, v}, nil)
}

func NewClient(board io.ReadWriteCloser) *Client {
	return &Client{
		board:                 board,
		responseChannels:      map[SysExCmd][]chan []byte{},
		sysExListenerChannels: map[SysExCmd]chan []byte{},
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.started = false
	return c.board.Close()
}

func (c *Client) Start() error {
	if c.started {
		return ErrAlreadyStarted
	}
	c.started = true

	firmChannel := make(chan []byte, 1)
	c.mu.Lock()
	c.responseChannels[SysExReportFirmware] = []chan []byte{firmChannel}
	c.mu.Unlock()
	report := c.parseReportFirmware(firmChannel)

	go func() {
		err := c.responseWatcher()
		if err != nil {
			panic(err)
		}
	}()

	fmt.Println("Firmware Info:", <-report)

	return nil
}

func (c *Client) write(payload []byte, withinMutex func()) error {
	// Cannot allow multiple writes at the same time.
	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Print("send:")
	for _, b := range payload {
		fmt.Printf(" 0x%0.2X", b)
	}
	fmt.Println()

	// Write to the board.
	_, err := c.board.Write(payload)
	if err != nil {
		return err
	}

	if withinMutex != nil {
		withinMutex()
	}

	return nil
}

func (c *Client) responseWatcher() (err error) {
	defer func() {
		if errors.Is(err, io.EOF) {
			err = ErrDeviceDisconnected
		}
	}()

	reader := bufio.NewReader(c.board)
	for {
		var data []byte
		b0, err := reader.ReadByte()
		if err != nil {
			return err
		}
		fmt.Printf("sys: 0x%0.2X %s\n", b0, MessageType(b0))

		switch MessageType(b0) {
		case ProtocolVersion:
			var version [2]byte
			_, err := reader.Read(version[:])
			if err != nil {
				return err
			}
			fmt.Printf("Protocol Version: 0x%0.2X 0x%0.2X\n", version[0], version[1])
		case StartSysEx:
			data, err = reader.ReadBytes(byte(EndSysEx))
			if err != nil {
				return err
			}

			if len(data) == 0 {
				return ErrNoDataRead
			}

			cmd := SysExCmd(data[0])
			fmt.Printf("msg: 0x%0.2X %s\n", data[0], cmd)

			switch {
			case cmd == SysExSerialDataV1:
				fallthrough
			case cmd == SysExSerialDataV2:
				return fmt.Errorf("%w: %s", ErrUnsupportedFeature, cmd)
			case cmd == SysExI2CReply:
				address := TwoByteToByte(data[1], data[2])
				ch, ok := c.i2cListeners[address]
				if !ok {
					return fmt.Errorf("%w: 0x%02X", ErrNoI2CListenerForAddress, address)
				}

				ch <- TwoByteRepresentationToByteSlice(data[3:])
			case c.sysExListenerChannels[cmd] != nil:
				c.sysExListenerChannels[cmd] <- data[1:]
			case len(c.responseChannels[cmd]) != 0:
				c.mu.Lock()
				resp := c.responseChannels[cmd][0]
				c.responseChannels[cmd] = c.responseChannels[cmd][1:]
				c.mu.Unlock()

				resp <- data[1 : len(data)-1]
				close(resp)
			default:
				str := ""
				if cmd == SysExStringData {
					str = TwoByteString(data[1:])
				} else {
					for _, b := range data[1:] {
						str += fmt.Sprintf("%d", b)
					}
				}

				return fmt.Errorf("%w: 0x%0.2X: %s", ErrUnexpectedSysExMessageTypeReceived, data[0], str)
			}
		default:
			return ErrInvalidMessageTypeStart
		}
	}
}

func (c *Client) SendReset() error {
	return c.write([]byte{byte(SystemReset)}, nil)
}

func (c *Client) SendSysEx(cmd SysExCmd, payload ...byte) (chan []byte, error) {
	// Create a response channel.
	var data chan []byte

	err := c.write(append([]byte{byte(StartSysEx), byte(cmd)}, append(payload, byte(EndSysEx))...), func() {
		// This assumes that SysEx commands of the same type are responded to in order.
		if resp, ok := commandResponseMap[cmd]; ok {
			data = make(chan []byte, 1)
			c.responseChannels[resp] = append(c.responseChannels[resp], data)
		}
	})
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *Client) CapabilityQuery() (chan CapabilityResponse, error) {
	future, err := c.SendSysEx(SysExCapabilityQuery)
	if err != nil {
		return nil, err
	}

	resp := c.parseCapabilityCommand(future)

	return resp, nil
}

func (c *Client) parseCapabilityCommand(future chan []byte) chan CapabilityResponse {
	resp := make(chan CapabilityResponse, 1)

	go func() {
		data := <-future
		var response = CapabilityResponse{
			SupportedPinModes: []map[PinMode]uint8{{}},
		}
		var pindex = 0
		for i := 0; i < len(data); {
			if data[i] == CapabilityResponsePinDelimiter {
				response.SupportedPinModes = append(response.SupportedPinModes, map[PinMode]uint8{})
				i += 1
				pindex++
			} else {
				response.SupportedPinModes[pindex][PinMode(data[i])] = data[i+1]
				i += 2
			}
		}

		c.cr = response
		resp <- response
		close(resp)
	}()

	return resp
}

func (c *Client) SendAnalogMappingQuery() (chan AnalogMappingResponse, error) {
	future, err := c.SendSysEx(SysExAnalogMappingQuery)
	if err != nil {
		return nil, err
	}

	resp := c.parseAnalogMappingQuery(future)

	return resp, nil
}

func (c *Client) parseAnalogMappingQuery(future chan []byte) chan AnalogMappingResponse {
	resp := make(chan AnalogMappingResponse, 1)

	go func() {
		data := <-future
		var response = AnalogMappingResponse{
			PinMapping: []uint8{},
		}
		for i := 0; i < len(data); i++ {
			if data[i] != CapabilityResponsePinDelimiter {
				response.PinMapping = append(response.PinMapping, uint8(i))
			}
		}

		c.amr = response
		resp <- response
		close(resp)
	}()

	return resp
}

func (c *Client) ExtendedReportAnalogPin(pin uint8, value int) error {
	if value > 0xFFFFFFFFFFFFFF {
		return fmt.Errorf("%w: 0x0 - 0xFFFFFFFFFFFFFF", ErrValueOutOfRange)
	}

	_, err := c.SendSysEx(SysExExtendedAnalog, pin, uint8(value), uint8(value>>7), uint8(value>>14))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) PinStateQuery(pin uint8) (chan PinStateResponse, error) {
	future, err := c.SendSysEx(SysExPinStateQuery, pin)
	if err != nil {
		return nil, err
	}

	resp := c.parsePinStateQuery(future)

	return resp, nil
}

func (c *Client) parsePinStateQuery(future chan []byte) chan PinStateResponse {
	resp := make(chan PinStateResponse, 1)

	go func() {
		data := <-future
		var ps = PinStateResponse{
			Pin:   data[0],
			Mode:  PinMode(data[1]),
			State: 0,
		}

		for i, b := range data[2:] {
			ps.State |= int(b << (i * 7))
		}

		resp <- ps
		close(resp)
	}()

	return resp
}

func (c *Client) ReportFirmware() (chan FirmwareReport, error) {
	future, err := c.SendSysEx(SysExReportFirmware)
	if err != nil {
		return nil, err
	}

	resp := c.parseReportFirmware(future)

	return resp, nil
}

func (c *Client) parseReportFirmware(future chan []byte) chan FirmwareReport {
	resp := make(chan FirmwareReport, 1)

	go func() {
		data := <-future
		var rc = FirmwareReport{
			Major: data[0],
			Minor: data[1],
			Name:  data[2:],
		}

		resp <- rc
		close(resp)
	}()

	return resp
}

func (c *Client) SetAnalogPinReporting(pin uint8, report bool) error {
	v := byte(0)
	if report {
		v = 1
	}

	return c.write([]byte{byte(ReportAnalogPin) | (pin & 0xF), v}, nil)
}

func (c *Client) SetDigitalPinReporting(pin uint8, report bool) error {
	return c.SetDigitalPortReporting(pin%8, report)
}

func (c *Client) SetDigitalPortReporting(port uint8, report bool) error {
	v := byte(0)
	if report {
		v = 1
	}

	return c.write([]byte{byte(ReportDigitalPort) | (port & 0xF), v}, nil)
}

func (c *Client) SetSamplingInterval(ms uint16) error {
	if ms > MaxUInt16 {
		return fmt.Errorf("%w: 0x0 - 0x%X", ErrValueOutOfRange, MaxUInt16)
	}
	return c.write([]byte{byte(SysExSamplingInterval), byte(ms), byte(ms >> 7)}, nil)
}

func (c *Client) SetDigitalMessageChannel(ch chan []byte) {

}

func (c *Client) SetAnalogMessageChannel(ch chan []byte) {

}

// This function only supports 7-bit I2C addresses
func (c *Client) SendI2CData(address uint8, restart bool, data []uint8) error {
	byte2 := byte(I2CModeWrite)
	if restart {
		byte2 &= I2CRestartTransmission
	}

	payload := append([]byte{address, byte2}, ByteSliceToTwoByteRepresentation(data)...)
	_, err := c.SendSysEx(SysExI2CRequest, payload...)
	return err
}

func (c *Client) SendI2CConfig(delayMicroseconds uint8) error {
	micLSB, micMSB := ByteToTwoByte(delayMicroseconds)
	_, err := c.SendSysEx(SysExI2CConfig, micLSB, micMSB)
	return err
}

func (c *Client) SetSerialMessageChannel(ch chan []byte) {
	if c.sysExListenerChannels[SysExSerialDataV2] != nil {
		close(c.sysExListenerChannels[SysExSerialDataV2])
	}

	c.sysExListenerChannels[SysExSerialDataV2] = ch
}

// This function only supports 7-bit I2C addresses
func (c *Client) SetI2CMessageChannel(address uint8, ch chan []byte) {
	if c.i2cListeners[address] != nil {
		close(c.i2cListeners[address])
	}

	c.i2cListeners[address] = ch
}
