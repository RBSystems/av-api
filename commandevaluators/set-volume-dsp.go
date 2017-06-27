package commandevaluators

import (
	"errors"
	"log"
	"strconv"

	"github.com/byuoitav/av-api/base"
	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/configuration-database-microservice/accessors"

	ei "github.com/byuoitav/event-router-microservice/eventinfrastructure"
)

type SetVolumeDSP struct{}

func (p *SetVolumeDSP) Evaluate(room base.PublicRoom) ([]base.ActionStructure, error) {

	log.Printf("Evaluating SetVolume command in DSP context...")

	eventInfo := ei.EventInfo{
		Type:         ei.CORESTATE,
		EventCause:   ei.USERINPUT,
		EventInfoKey: "volume",
	}

	var actions []base.ActionStructure

	if room.Volume != nil {

		log.Printf("Room-wide request detected")

		dsp, err := dbo.GetDevicesByBuildingAndRoomAndRole(room.Building, room.Room, "DSP")
		if err != nil {
			errorMessage := "Could not find DSP in room: " + room.Room + " in building: " + room.Building + " : " + err.Error()
			log.Printf(errorMessage)
			return []base.ActionStructure{}, errors.New(errorMessage)
		}

		if len(dsp) != 1 {
			errorMessage := "Invalid number of DSP devices found in room: " + room.Room + " in building: " + room.Building
			log.Printf(errorMessage)
			return []base.ActionStructure{}, errors.New(errorMessage)
		}

		action, err := GetDSPMediaVolumeAction(dsp[0], room, eventInfo)
		if err != nil {
			errorMessage := "Could not generate DSP media action for DSP: " + dsp[0].Name + " : " + err.Error()
			log.Printf(errorMessage)
			return []base.ActionStructure{}, errors.New(errorMessage)
		}

		actions = append(actions, action)
	}

	if len(room.AudioDevices) > 0 {

		for _, audioDevice := range room.AudioDevices {

			device, err := dbo.GetDeviceByName(room.Building, room.Room, audioDevice.Name)
			if err != nil {
				log.Printf("Error getting device %s from database: %s", audioDevice.Name, err.Error())
			}

			if device.HasRole("Microphone") && audioDevice.Volume != nil {

				action, err := GetMicVolumeAction(device, room, eventInfo)
				if err != nil {
					return []base.ActionStructure{}, err
				}

				actions = append(actions, action)

			} else if device.HasRole("DSP") && device.Volume != nil {

				action, err := GetDSPMediaVolumeAction(device, room, eventInfo)
				if err != nil {
					return []base.ActionStructure{}, err
				}

				actions = append(actions, action)

			} else if device.HasRole("AudioOut") && device.Volume != nil {

				action, err := GetDisplayVolumeAction(device, room, eventInfo)
				if err != nil {
					return []base.ActionStructure{}, err
				}

				actions = append(actions, action)

			} else { //bad device
				errorMessage := "Cannot set volume of device " + device.Name
				log.Printf(errorMessage)
				return []base.ActionStructure{}, errors.New(errorMessage)
			}
		}
	}

	log.Printf("%s actions generated.", len(actions))
	log.Printf("Evaluation complete.")
	return actions, nil
}

func (p *SetVolumeDSP) Validate(action base.ActionStructure) (err error) {
	maximum := 100
	minimum := 0

	level, err := strconv.Atoi(action.Parameters["level"])
	if err != nil {
		return err
	}

	if level > maximum || level < minimum {
		log.Printf("ERROR. %v is an invalid volume level for %s", action.Parameters["level"], action.Device.Name)
		return errors.New(action.Action + " is an invalid command for " + action.Device.Name)
	}

	return
}

func (p *SetVolumeDSP) GetIncompatibleCommands() (incompatibleActions []string) {
	return nil
}

func GetGeneralVolumeRequestActionsDSP(room base.PublicRoom, eventInfo ei.EventInfo) ([]base.ActionStructure, error) {

	log.Printf("Generating actions for room-wide \"SetVolume\" request")

	var actions []base.ActionStructure

	dsp, err := dbo.GetDevicesByBuildingAndRoomAndRole(room.Building, room.Room, "DSP")
	if err != nil {
		log.Printf("Error getting devices %s", err.Error)
		return []base.ActionStructure{}, err
	}

	action, err := GetDSPMediaVolumeAction(dsp[0], room, eventInfo)
	if err != nil {
		errorMessage := "Could not generate action corresponding to general mute request in room " + room.Room + ", building " + room.Building + ": " + err.Error()
		log.Printf(errorMessage)
		return []base.ActionStructure{}, errors.New(errorMessage)
	}

	actions = append(actions, action)

	audioDevices, err := dbo.GetDevicesByBuildingAndRoomAndRole(room.Building, room.Room, "AudioOut")
	if err != nil {
		log.Printf("Error getting devices %s", err.Error())
		return []base.ActionStructure{}, err
	}

	for _, device := range audioDevices {

		action, err := GetDisplayVolumeAction(device, room, eventInfo)
		if err != nil {
			errorMessage := "Could not generate mute action for display " + device.Name + " in room " + room.Room + ", building " + room.Building + ": " + err.Error()
			log.Printf(errorMessage)
			return []base.ActionStructure{}, errors.New(errorMessage)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

//we assume microphones are only connected to a DSP
//commands regarding microphones are only issued to DSP
func GetMicVolumeAction(mic accessors.Device, room base.PublicRoom, eventInfo ei.EventInfo) (base.ActionStructure, error) {

	log.Printf("Identified microphone volume request")

	parameters := make(map[string]string)

	eventInfo.EventInfoValue = string(*mic.Volume)
	parameters["volume"] = string(*mic.Volume)
	parameters["input"] = mic.Ports[0].Name

	dsp, err := dbo.GetDeviceByName(room.Building, room.Room, mic.Ports[0].Destination)
	if err != nil {
		errorMessage := "Could not get DSP corresponding to mic " + mic.Name + ": " + err.Error()
		log.Printf(errorMessage)
		return base.ActionStructure{}, errors.New(errorMessage)
	}

	return base.ActionStructure{
		Action:              "SetVolume",
		GeneratingEvaluator: "SetVolumeDSP",
		Device:              dsp,
		DeviceSpecific:      true,
		EventLog:            []ei.EventInfo{eventInfo},
		Parameters:          parameters,
	}, nil

}

func GetDSPMediaVolumeAction(device accessors.Device, room base.PublicRoom, eventInfo ei.EventInfo) (base.ActionStructure, error) { //commands are issued to whatever port doesn't have a mic connected

	log.Printf("Generating action for command SetVolume on media routed through DSP")

	parameters := make(map[string]string)
	parameters["volume"] = string(*device.Volume)

	eventInfo.EventInfoValue = string(*device.Volume)

	for _, port := range device.Ports {

		sourceDevice, err := dbo.GetDeviceByName(room.Building, room.Room, port.Source)
		if err != nil {
			errorMessage := "Could not get device " + port.Source + " from database: " + err.Error()
			log.Printf(errorMessage)
			return base.ActionStructure{}, errors.New(errorMessage)
		}

		if !(sourceDevice.HasRole("Microphone")) {

			parameters["input"] = port.Name
			action := base.ActionStructure{
				Action:              "SetVolume",
				GeneratingEvaluator: "SetVolumeDSP",
				Device:              device,
				DeviceSpecific:      true,
				EventLog:            []ei.EventInfo{eventInfo},
			}

			return action, nil
		}
	}

	return base.ActionStructure{}, nil

}

func GetDisplayVolumeAction(device accessors.Device, room base.PublicRoom, eventInfo ei.EventInfo) (base.ActionStructure, error) { //commands are issued to devices, e.g. they aren't connected to the DSP

	log.Printf("Generating action for SetVolume on device %s external to DSP", device.Name)

	parameters := make(map[string]string)

	eventInfo.EventInfoValue = strconv.Itoa(*device.Volume)
	parameters["volume"] = string(*device.Volume)

	action := base.ActionStructure{
		Action:              "SetVolume",
		GeneratingEvaluator: "SetVolumeDSP",
		Device:              device,
		DeviceSpecific:      true,
		EventLog:            []ei.EventInfo{eventInfo},
		Parameters:          parameters,
	}

	return action, nil
}
