package state

import (
	"log"

	"github.com/byuoitav/av-api/base"
	"github.com/byuoitav/av-api/dbo"
	"github.com/byuoitav/av-api/statusevaluators"
	"github.com/byuoitav/common/nerr"
	"github.com/fatih/color"
)

func GetRoomState(building string, roomName string) (base.PublicRoom, *nerr.E) {

	color.Set(color.FgHiCyan, color.Bold)
	log.L.Debugf("[state] getting room state...")
	color.Unset()

	room, err := dbo.GetRoomByInfo(building, roomName)
	if err != nil {
		return base.PublicRoom{}, err.Addf("Couldn't get room from database.")
	}

	//we get the number of actions generated
	commands, count, err := GenerateStatusCommands(room, statusevaluators.STATUS_EVALUATORS)
	if err != nil {
		return base.PublicRoom{}, err.Addf("Couldn't generate status commands.")
	}

	responses, err := RunStatusCommands(commands)
	if err != nil {
		return base.PublicRoom{}, err.Addf("Error running status commands.")
	}

	roomStatus, err := EvaluateResponses(responses, count)
	if err != nil {
		return base.PublicRoom{}, err.Addf("Error evaluating status responses")
	}

	roomStatus.Building = building
	roomStatus.Room = roomName

	color.Set(color.FgHiGreen, color.Bold)
	log.L.Debugf("[state] successfully retrieved room state")
	color.Unset()

	return roomStatus, nil
}

func SetRoomState(target base.PublicRoom, requestor string) (base.PublicRoom, *nerr.E) {

	log.L.Debugf("[state] setting room state...")

	room, err := dbo.GetRoomByInfo(target.Building, target.Room)
	if err != nil {
		return base.PublicRoom{}, err.Addf("Couldn't get room from database.")
	}

	//so here we need to know how many things we're actually expecting.
	actions, count, err := GenerateActions(room, target, requestor)
	if err != nil {
		return base.PublicRoom{}, err
	}

	responses, err := ExecuteActions(actions, requestor)
	if err != nil {
		return base.PublicRoom{}, err
	}

	//here's where we then pass that information through so that we can make a decent decision.
	report, err := EvaluateResponses(responses, count)
	if err != nil {
		return base.PublicRoom{}, err
	}

	report.Building = target.Building
	report.Room = target.Room

	color.Set(color.FgHiGreen, color.Bold)
	log.L.Debugf("[state] successfully set room state")
	color.Unset()

	return report, nil
}
