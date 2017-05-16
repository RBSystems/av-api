package status

import (
	"log"

	"github.com/byuoitav/configuration-database-microservice/accessors"
)

type PowerDefault struct {
}

//when querying power, we care about every device
func (p *PowerDefault) GetDevices(room accessors.Room) ([]accessors.Device, error) {
	return room.Devices, nil
}

func (p *PowerDefault) GenerateCommands(devices []accessors.Device) ([]StatusCommand, error) {

	log.Printf("Generating default power commands...")

	var output []StatusCommand

	//iterate over each device
	for _, device := range devices {

		log.Printf("Considering device: %s", device.Name)

		for _, command := range device.Commands {

			if command.Name == "STATUS_Power" {

				//every power command needs an address parameter
				parameters := make(map[string]string)
				parameters["address"] = device.Address

				//build destination device
				var destinationDevice DestinationDevice
				for _, role := range device.Roles {

					if role == "AudioOut" {
						destinationDevice.AudioDevice = true
					}

					if role == "VideoOut" {
						destinationDevice.Display = true
					}

				}
				destinationDevice.Device = device

				log.Printf("Adding command: %s to action list with device %s", command.Name, device.Name)
				output = append(output, StatusCommand{
					Action:            command,
					Device:            device,
					Parameters:        parameters,
					DestinationDevice: destinationDevice,
				})

			}

		}

	}
	return output, nil
}
