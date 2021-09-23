package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"

	"github.com/imle/go-firmata"
)

func main() {
	board, err := serial.OpenPort(&serial.Config{
		Name: "/dev/cu.usbmodem14601",
		Baud: 57600,
	})
	handleError(err)
	defer board.Close()

	c := firmata.NewClient(board)

	err = c.SendReset()
	handleError(err)

	err = c.Start()
	handleError(err)

	analogMappingQuery, err := c.SendAnalogMappingQuery()
	handleError(err)

	capabilityQuery, err := c.CapabilityQuery()
	handleError(err)

	fmt.Println(<-analogMappingQuery)
	fmt.Println(<-capabilityQuery)

	psResp, err := c.PinStateQuery(17)
	handleError(err)
	fmt.Println(<-psResp)

	err = c.SetPinMode(17, firmata.PinModeDigitalOutput)
	handleError(err)

	psResp, err = c.PinStateQuery(17)
	handleError(err)
	fmt.Println(<-psResp)

	v := true

	for true {
		time.Sleep(time.Second * 1)

		err = c.SetDigitalPinValue(17, v)
		handleError(err)

		v = !v

		psResp, err = c.PinStateQuery(17)
		handleError(err)
		fmt.Println(<-psResp)
	}
}

func handleError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
