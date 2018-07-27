package state

import (
	"fmt"
	"strings"
	"sync"

	"github.com/byuoitav/av-api/actionreconcilers"
	"github.com/byuoitav/av-api/base"
	ce "github.com/byuoitav/av-api/commandevaluators"
	se "github.com/byuoitav/av-api/statusevaluators"
	"github.com/byuoitav/common/log"
	"github.com/byuoitav/common/nerr"
	"github.com/byuoitav/configuration-database-microservice/structs"
	"github.com/fatih/color"
)

//for each command in the configuration, evaluate and validate.
func GenerateActions(dbRoom structs.Room, bodyRoom base.PublicRoom, requestor string) ([]base.ActionStructure, int, *nerr.E) {

	log.L.Infof("Generating actions...")

	var count int

	var output []base.ActionStructure
	for _, evaluator := range dbRoom.Configuration.Evaluators {

		if strings.Contains(evaluator.EvaluatorKey, "STATUS") {
			continue
		}

		curEvaluator := ce.EVALUATORS[evaluator.EvaluatorKey]
		if curEvaluator == nil {
			return []base.ActionStructure{}, 0, nerr.Create(fmt.Sprintf("no evaluator corresponding to key: %s", evaluator.EvaluatorKey), "invalid-config")
		}

		actions, c, err := curEvaluator.Evaluate(bodyRoom, requestor)
		if err != nil {
			return []base.ActionStructure{}, 0, err.Addf("Couldn't generate actions for evaluator %v", evaluator.EvaluatorKey)
		}

		for _, action := range actions {
			err := curEvaluator.Validate(action)
			if err != nil {
				msg := fmt.Sprintf("action %s not valid with evaluator %s: %s", action.Action, evaluator.EvaluatorKey, err.Error())

				log.L.Warnf(msg)
				return []base.ActionStructure{}, 0, nerr.Create(msg, "invalid-action")
			}

			// Provide a map from the generating evaluator to the generated action in
			// case they want to use the Incompatable actions in the reconcilers.
			action.GeneratingEvaluator = evaluator.EvaluatorKey
		}

		output = append(output, actions...)
		count += c
	}

	log.L.Infof("Generated %v total actions.", len(output))

	batches, count, err := ReconcileActions(dbRoom, output, count)

	return batches, count, err
}

//produces a DAG
func ReconcileActions(room structs.Room, actions []base.ActionStructure, inCount int) (batches []base.ActionStructure, count int, err *nerr.E) {

	log.L.Debugf("Reconciling actions...")

	//Initialize map of strings to commandevaluators
	reconcilers := actionreconcilers.Init()

	curReconciler := reconcilers[room.Configuration.RoomKey]
	if curReconciler == nil {
		err = nerr.Create(fmt.Sprintf("no reconciler corresponding to key: %s ", room.Configuration.RoomKey), "invalid-config")
		return
	}

	batches, count, err = curReconciler.Reconcile(actions, inCount)
	if err != nil {
		return
	}

	log.L.Debugf("Done reconciling actions.")

	return
}

//@pre TODO DestinationDevice field is populated for every action!!
//ExecuteActions carries out the actions defined in the struct
func ExecuteActions(DAG []base.ActionStructure, requestor string) ([]se.StatusResponse, *nerr.E) {

	log.L.Debugf("%s", color.HiBlueString("[state] executing actions..."))

	if len(DAG) == 0 {
		return []se.StatusResponse{}, nerr.Create("No actions to execute", "no-op")
	}

	var output []se.StatusResponse

	responses := make(chan se.StatusResponse, len(DAG))
	var done sync.WaitGroup

	for _, child := range DAG[0].Children {

		done.Add(1)
		go ExecuteAction(*child, responses, &done, requestor)
	}

	log.L.Debugf("[state] waiting for responses...")
	done.Wait()

	log.L.Debugf("[state] done executing actions, closing channel...")
	close(responses)

	if len(responses) < len(DAG)-1 {
		log.L.Debugf("%s", color.HiRedString("[*nerr.E] expecting %v responses, found %v", len(DAG), len(responses)))
	}

	for response := range responses {
		output = append(output, response)
	}

	log.L.Debugf("%s", color.HiBlueString("[state] done executing actions"))

	return output, nil
}

//builds a status response
func ExecuteAction(action base.ActionStructure, responses chan<- se.StatusResponse, control *sync.WaitGroup, requestor string) {

	log.L.Debugf("[state] Executing action %s against device %s...", action.Action, action.Device.Name)

	if action.Overridden {
		log.L.Debugf("[state] Action %s on device %s have been overridden. Continuing.",
			action.Action, action.Device.Name)
		control.Done()
		return
	}

	has, cmd := ce.CheckCommands(action.Device.Commands, action.Action)
	if !has {
		errstr := fmt.Sprintf("[state] Error retrieving the command %s for device %s.", action.Action, action.Device.GetFullName())
		log.L.Debugf(errstr)
		PublishError(errstr, action, requestor)
		control.Done()
		return
	}

	endpoint := ReplaceIPAddressEndpoint(cmd.Endpoint.Path, action.Device.Address)
	endpoint, err := ReplaceParameters(endpoint, action.Parameters)
	if err != nil {
		msg := fmt.Sprintf("Error building endpoint for command %s against device %s: %s", action.Action, action.Device.GetFullName(), err.Error())
		log.L.Debugf("%s", color.HiRedString("[state] %s", msg))
		PublishError(msg, action, requestor)
		control.Done()
		return
	}

	//Execute the command.
	status := ExecuteCommand(action, cmd, endpoint, requestor)

	log.L.Debugf("[state] Writing response to channel...")
	responses <- status
	log.L.Debugf("[state] microservice reported status: %v", status.Status)

	for _, child := range action.Children {

		log.L.Debugf("[state] found child: %s. Executing...", child.Action)

		control.Add(1)
		go ExecuteAction(*child, responses, control, requestor)
	}

	control.Done()
}

//this is where we decide which status evaluator is used to evalutate the resultant status of a command that sets state
var SET_STATE_STATUS_EVALUATORS = map[string]string{

	"PowerOnDefault":                 "STATUS_PowerDefault",
	"StandbyDefault":                 "STATUS_PowerDefault",
	"ChangeVideoInputDefault":        "STATUS_InputDefault",
	"ChangeAudioInputDefault":        "STATUS_InputDefault",
	"ChangeVideoInputVideoSwitcher":  "STATUS_InputVideoSwitcher",
	"ChangeVideoInputTieredSwitcher": "STATUS_InputVideoSwitcher",
	"BlankDisplayDefault":            "STATUS_BlankedDefault",
	"UnBlankDisplayDefault":          "STATUS_BlankedDefault",
	"MuteDefault":                    "STATUS_MutedDefault",
	"UnMuteDefault":                  "STATUS_MutedDefault",
	"SetVolumeDefault":               "STATUS_VolumeDefault",
	"SetVolumeTecLite":               "STATUS_VolumeDefault",
	"MuteDSP":                        "STATUS_MutedDSP",
	"UnmuteDSP":                      "STATUS_MutedDSP",
	"SetVolumeDSP":                   "STATUS_VolumeDSP",
}
