package inputgraph

import (
	"errors"
	"fmt"
	"log"

	"github.com/byuoitav/configuration-database-microservice/structs"
	"github.com/fatih/color"
)

var debug = true

type InputGraph struct {
	Nodes        []*Node
	AdjecencyMap map[string][]string
	DeviceMap    map[string]*Node
}

type Node struct {
	ID     string
	Device structs.Device
}

func BuildGraph(devs []structs.Device) (InputGraph, error) {

	ig := InputGraph{
		AdjecencyMap: make(map[string][]string),
		DeviceMap:    make(map[string]*Node),
		Nodes:        []*Node{},
	}

	//we go through and build our graph
	for d := range devs {

		if _, ok := ig.DeviceMap[devs[d].Name]; !ok {
			newNode := Node{ID: devs[d].Name, Device: devs[d]}
			ig.Nodes = append(ig.Nodes, &newNode)

			ig.DeviceMap[devs[d].Name] = &newNode
		}

		for _, port := range devs[d].Ports {
			if debug {
				log.Printf("Addding %v to the adjecency for %v based on port %v", port.Source, port.Destination, port.Name)
			}
			//we add the entry in the adjacency map
			if _, ok := ig.AdjecencyMap[port.Destination]; ok {
				ig.AdjecencyMap[port.Destination] = append(ig.AdjecencyMap[port.Destination], port.Source)
			} else {
				ig.AdjecencyMap[port.Destination] = []string{port.Source}
			}
		}
	}

	//TODO: do we need to go through and check the Adjecency maps for duplicates?

	return ig, nil
}

//where deviceA is the sink and deviceB is the source
func CheckReachability(deviceA, deviceB string, ig InputGraph) (bool, []Node, error) {
	if debug {
		log.Printf("looking for a path from %v to %v", deviceA, deviceB)

	}
	//check and make sure that both of the devices are actually a part of the graph

	if _, ok := ig.DeviceMap[deviceA]; !ok {
		msg := fmt.Sprintf("device %v is not part of the graph", deviceA)

		log.Printf(color.HiRedString(msg))

		return false, []Node{}, errors.New(msg)
	}

	if _, ok := ig.DeviceMap[deviceB]; !ok {
		msg := fmt.Sprintf("device %v is not part of the graph", deviceA)

		log.Printf(color.HiRedString(msg))

		return false, []Node{}, errors.New(msg)
	}

	//now we need to check to see if we can get from a to b. We're gonna use a BFS
	frontier := make(chan string, len(ig.Nodes))
	visited := make(map[string]bool)
	path := make(map[string]string)

	//put in our first state
	frontier <- deviceA

	visited[deviceA] = true

	for {
		select {
		case cur := <-frontier:
			if debug {
				log.Printf("Evaluating %v", cur)
			}
			if cur == deviceB {
				if debug {
					log.Printf("Destination reached.", cur)
				}
				dev := cur

				toReturn := []Node{}
				toReturn = append(toReturn, *ig.DeviceMap[dev])
				if debug {
					log.Printf("First Hop: %v -> %v", dev, path[dev])
				}

				dev, ok := path[dev]

				count := 0
				for ok {
					if count > len(path) {
						msg := "Circular path detected: returning"
						log.Printf(color.HiRedString(msg))

						return false, []Node{}, errors.New(msg)
					}
					if debug {
						log.Printf("Next hop: %v -> %v", dev, path[dev])
					}

					toReturn = append(toReturn, *ig.DeviceMap[dev])

					dev, ok = path[dev]
					count++

				}
				//get our path and return it
				return true, toReturn, nil
			}

			for _, next := range ig.AdjecencyMap[cur] {
				if _, ok := path[next]; ok || next == deviceA {
					continue
				}

				path[next] = cur
				if debug {

					log.Printf("Path from %v to %v, adding %v to frontier", cur, next, next)
					log.Printf("Path as it stands is: ")

					curDev := next
					dev, ok := path[curDev]
					for ok {
						log.Printf("%v -> %v", curDev, dev)
						curDev = dev
						dev, ok = path[curDev]
					}
				} //END DEBUG
				frontier <- next
			}
		default:
			if debug {
				log.Printf("No path found")
			}
			return false, []Node{}, nil
		}
	}
}
