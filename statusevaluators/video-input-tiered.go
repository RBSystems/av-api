package statusevaluators

import (
	"strings"
	"time"

	"github.com/byuoitav/av-api/base"
	"github.com/byuoitav/av-api/statusevaluators/pathfinder"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/structs"
	"github.com/fatih/color"
)

// InputTieredSwitcherEvaluator is a constant variable for the name of the evaluator.
const InputTieredSwitcherEvaluator = "STATUS_Tiered_Switching"

// InputTieredSwitcher implements the StatusEvaluator struct.
type InputTieredSwitcher struct {
}

// GetDevices returns a list of devices in the given room.
func (p *InputTieredSwitcher) GetDevices(room structs.Room) ([]structs.Device, error) {
	return room.Devices, nil
}

// GenerateCommands generates a list of commands for the given devices.
func (p *InputTieredSwitcher) GenerateCommands(devs []structs.Device) ([]StatusCommand, int, error) {
	//look at all the output devices and switchers in the room. we need to generate a status input for every port on every video switcher and every output device.

	//TODO use the 'bulk' endpoints and parse that. In that case there'd be two different paths, one for the change input and one for the get status.

	callbackEngine := &TieredSwitcherCallback{}
	toReturn := []StatusCommand{}
	var count int

	for _, d := range devs {
		isVS := structs.HasRole(d, "VideoSwitcher")
		cmd := d.GetCommandByName("STATUS_Input")
		if len(cmd.ID) == 0 {
			continue
		}
		if (!d.Type.Output && !isVS) || structs.HasRole(d, "Microphone") || structs.HasRole(d, "DSP") { //we don't care about it
			continue
		}

		//validate it has the command
		if len(cmd.ID) == 0 {
			log.L.Error(color.HiRedString("[error] No input command for device %v...", d.Name))
			continue
		}

		if isVS {
			log.L.Info("[statusevals] Identified video switcher, generating commands...")
			//we need to generate commands for every output port

			for _, p := range d.Ports {
				//if it's an OUT port
				if strings.Contains(p.ID, "OUT") {
					//we need to strip the value

					name := strings.Replace(p.ID, "OUT", "", -1)

					params := make(map[string]string)
					params["address"] = d.Address
					params["port"] = name

					//this is where we'd add the callback
					toReturn = append(toReturn, StatusCommand{
						Action:     cmd,
						Device:     d,
						Generator:  InputTieredSwitcherEvaluator,
						Parameters: params,
						Callback:   callbackEngine.Callback,
					})
				}
			}
			//we've finished with the switch
			continue
		} //now we deal with the output devices, which is pretty basic
		params := make(map[string]string)
		params["address"] = d.Address

		toReturn = append(toReturn, StatusCommand{
			Action: cmd,
			Device: d,
			DestinationDevice: base.DestinationDevice{
				Device:      d,
				AudioDevice: structs.HasRole(d, "AudioOut"),
				Display:     structs.HasRole(d, "VideoOut"),
			},
			Generator:  InputTieredSwitcherEvaluator,
			Parameters: params,
			Callback:   callbackEngine.Callback,
		})
		//we only count the number of output devices
		count++

	}

	callbackEngine.InChan = make(chan base.StatusPackage, len(toReturn))
	callbackEngine.ExpectedCount = count
	callbackEngine.ExpectedActionCount = len(toReturn)
	callbackEngine.Devices = devs

	go callbackEngine.StartAggregator()

	for _, a := range toReturn {
		log.L.Infof(color.HiYellowString("%v, %v, %v", a.Action, a.Device.Name, a.Parameters))
	}

	return toReturn, count, nil
}

// EvaluateResponse processes the response information that is given.
func (p *InputTieredSwitcher) EvaluateResponse(str string, face interface{}, dev structs.Device, destDev base.DestinationDevice) (string, interface{}, error) {
	return "", nil, nil

}

// TieredSwitcherCallback defines the callback information for the tiered switching commands and responses.
type TieredSwitcherCallback struct {
	InChan              chan base.StatusPackage
	OutChan             chan<- base.StatusPackage
	Devices             []structs.Device
	ExpectedCount       int
	ExpectedActionCount int
}

// Callback begins the callback process...
func (p *TieredSwitcherCallback) Callback(sp base.StatusPackage, c chan<- base.StatusPackage) error {
	log.L.Info(color.HiYellowString("[callback] calling"))
	log.L.Infof(color.HiYellowString("[callback] Device: %v", sp.Device.ID))
	log.L.Infof(color.HiYellowString("[callback] Dest Device: %v", sp.Dest.ID))
	log.L.Infof(color.HiYellowString("[callback] Key: %v", sp.Key))
	log.L.Infof(color.HiYellowString("[callback] Value: %v", sp.Value))

	log.L.Infof(color.HiYellowString("[callback] ExpectedCount: %v", p.ExpectedCount))
	log.L.Infof(color.HiYellowString("[callback] ExpectedActionCount: %v", p.ExpectedActionCount))

	//we pass down the the aggregator that was started before
	p.OutChan = c
	p.InChan <- sp

	return nil
}

func (p *TieredSwitcherCallback) getDeviceByID(dev string) structs.Device {
	for d := range p.Devices {
		if p.Devices[d].ID == dev {
			return p.Devices[d]
		}
	}
	return structs.Device{}
}

// GetInputPaths generates a directed graph of the tiered switching layout.
func (p *TieredSwitcherCallback) GetInputPaths(pathfinder pathfinder.SignalPathfinder) {
	//we need to get the status that we can - odds are good we're in a room where the displays are off.

	//how to traverse the graph for some of the output devices - we check to see if the output device is connected somehow - and we report where it got to.

	inputMap, err := pathfinder.GetInputs()
	if err != nil {
		log.L.Error("Error getting the inputs")
		return
	}

	for k, v := range inputMap {
		outDev := p.getDeviceByID(k)
		if len(outDev.ID) == 0 {
			log.L.Warnf("No device by name %v in the device list for the callback", k)
		}

		destDev := base.DestinationDevice{
			Device:      outDev,
			AudioDevice: structs.HasRole(outDev, "AudioOut"),
			Display:     structs.HasRole(outDev, "VideoOut"),
		}
		log.L.Infof(color.HiYellowString("[callback] Sending input %v -> %v", v.Name, k))

		p.OutChan <- base.StatusPackage{
			Dest:  destDev,
			Key:   "input",
			Value: v.Name,
		}
	}
	log.L.Info(color.HiYellowString("[callback] Done with evaluation. Closing."))
	return
}

// StartAggregator starts the aggregator...I guess haha...
func (p *TieredSwitcherCallback) StartAggregator() {
	log.L.Info(color.HiYellowString("[callback] Starting aggregator."))
	started := false

	t := time.NewTimer(0)
	<-t.C
	pathfinder := pathfinder.InitializeSignalPathfinder(p.Devices, p.ExpectedActionCount)

	for {
		select {
		case <-t.C:
			//we're timed out
			log.L.Warn(color.HiYellowString("[callback] Timeout."))
			p.GetInputPaths(pathfinder)
			return

		case val := <-p.InChan:
			log.L.Info(color.HiYellowString("[callback] Received Information, adding an edge: %v %v", val.Device.Name, val.Value))
			//start our timeout
			if !started {
				log.L.Info("[callback] Started aggregator timeout")
				started = true
				t.Reset(500 * time.Millisecond)
			}

			//we need to start our graph, then check if we have any completed paths
			ready := pathfinder.AddEdge(val.Device, val.Value.(string))
			if ready {
				log.L.Info(color.HiYellowString("[callback] All Information received."))
				p.GetInputPaths(pathfinder)
				return
			}
		}
	}
}
