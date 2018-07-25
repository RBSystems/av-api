package commandevaluators

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/byuoitav/common/log"

	"github.com/byuoitav/av-api/base"
	"github.com/byuoitav/common/db"
	"github.com/byuoitav/common/events"
	"github.com/byuoitav/common/structs"
)

// SetVolumeDefault implements the CommandEvaluation struct.
type SetVolumeDefault struct {
}

//Evaluate checks for a volume for the entire room or the volume of a specific device
func (*SetVolumeDefault) Evaluate(room base.PublicRoom, requestor string) ([]base.ActionStructure, int, error) {

	var actions []base.ActionStructure

	eventInfo := events.EventInfo{
		Type:         events.CORESTATE,
		EventCause:   events.USERINPUT,
		EventInfoKey: "volume",
		Requestor:    requestor,
	}

	destination := base.DestinationDevice{
		AudioDevice: true,
	}

	// general room volume
	if room.Volume != nil {

		log.L.Info("[command_evaluators] General volume request detected.")

		roomID := fmt.Sprintf("%v-%v", room.Building, room.Room)
		devices, err := db.GetDB().GetDevicesByRoomAndRole(roomID, "AudioOut")
		if err != nil {
			return []base.ActionStructure{}, 0, err
		}

		for _, device := range devices {

			if device.Type.Output {

				parameters := make(map[string]string)
				parameters["level"] = fmt.Sprintf("%v", *room.Volume)

				eventInfo.EventInfoValue = fmt.Sprintf("%v", *room.Volume)
				eventInfo.Device = device.Name
				destination.Device = device

				if structs.HasRole(device, "VideoOut") {
					destination.Display = true
				}

				actions = append(actions, base.ActionStructure{
					Action:              "SetVolume",
					Parameters:          parameters,
					GeneratingEvaluator: "SetVolumeDefault",
					Device:              device,
					DestinationDevice:   destination,
					DeviceSpecific:      false,
					EventLog:            []events.EventInfo{eventInfo},
				})

			}

		}

	}

	//identify devices in request body
	if len(room.AudioDevices) != 0 {

		log.L.Info("[command_evaluators] Device specific request detected. Scanning devices")

		for _, audioDevice := range room.AudioDevices {
			// create actions based on request

			if audioDevice.Volume != nil {
				log.L.Info("[command_evaluators] Adding device %+v", audioDevice.Name)

				deviceID := fmt.Sprintf("%v-%v-%v", room.Building, room.Room, audioDevice.Name)
				device, err := db.GetDB().GetDevice(deviceID)
				if err != nil {
					return []base.ActionStructure{}, 0, err
				}

				parameters := make(map[string]string)
				parameters["level"] = fmt.Sprintf("%v", *audioDevice.Volume)
				log.L.Info("[command_evaluators] %+v", parameters)

				eventInfo.EventInfoValue = fmt.Sprintf("%v", *audioDevice.Volume)
				eventInfo.Device = device.Name
				destination.Device = device

				if structs.HasRole(device, "VideoOut") {
					destination.Display = true
				}

				actions = append(actions, base.ActionStructure{
					Action:              "SetVolume",
					GeneratingEvaluator: "SetVolumeDefault",
					Device:              device,
					DestinationDevice:   destination,
					DeviceSpecific:      true,
					Parameters:          parameters,
					EventLog:            []events.EventInfo{eventInfo},
				})

			}

		}

	}

	log.L.Infof("[command_evaluators] %v actions generated.", len(actions))
	log.L.Info("[command_evaluators] Evaluation complete.")

	return actions, len(actions), nil
}

func validateSetVolumeMaxMin(action base.ActionStructure, maximum int, minimum int) error {
	level, err := strconv.Atoi(action.Parameters["level"])
	if err != nil {
		return err
	}

	if level > maximum || level < minimum {
		msg := fmt.Sprintf("[command_evaluators] ERROR. %v is an invalid volume level for %s", action.Parameters["level"], action.Device.Name)
		log.L.Error(msg)
		return errors.New(msg)
	}
	return nil
}

//Validate returns an error if the volume is greater than 100 or less than 0
func (p *SetVolumeDefault) Validate(action base.ActionStructure) error {
	maximum := 100
	minimum := 0

	return validateSetVolumeMaxMin(action, maximum, minimum)

}

//GetIncompatibleCommands returns a string array of commands incompatible with setting the volume
func (p *SetVolumeDefault) GetIncompatibleCommands() []string {
	return nil
}
