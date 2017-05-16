package status

import (
	"log"

	"github.com/byuoitav/configuration-database-microservice/accessors"
)

type VolumeDefault struct {
}

func (p *VolumeDefault) GetDevices(room accessors.Room) ([]accessors.Device, error) {

	var output []accessors.Device

	for _, device := range room.Devices {

		for _, role := range device.Roles {

			if role == "AudioOut" {

				output = append(output, device)

			}

		}

	}

	return output, nil
}

func (p *VolumeDefault) GenerateCommands(devices []accessors.Device) ([]StatusCommand, error) {

	log.Printf("Generating default volume commands...")

	var output []StatusCommand

	for _, device := range devices {

		log.Printf("Considering device: %s", device.Name)

		for _, command := range device.Commands {

			if command.Name == "STATUS_Volume" {

				parameters := make(map[string]string)
				parameters["address"] = device.Address

				destination := DestinationDevice{AudioDevice: true}

				destination.ID = device.ID
				destination.Name = device.Name

				log.Printf("Adding command: %s to action list for device %s", command.Name, device.Name)

				output = append(output, StatusCommand{
					Action:            command,
					Device:            device,
					Parameters:        parameters,
					DestinationDevice: destination,
				})

			}

		}

	}

	return []StatusCommand{}, nil
}
