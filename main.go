/* blackbird logs in to the linux machines in parallel and will query their current
ram, cpu and disk utilizations and provides log/mail notifications if configured.
it can access the nodes directly from this machine or via a ssh gateway (tunnel).
*/
package main

import (
	"blackbird/pkgs/ioreader"
	"blackbird/pkgs/retriever"
	"log"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

func main() {

	//config.json is the file to be configured to determine the script execution steps, mail and sshgw addresses, worker quantity, etc.
	configFileName := "config.json"

	//configFilePath is used to change the config file's path to an os specific address.
	configFilePath := filepath.Join("conf", configFileName)

	//Config reads the json file and returns a struct which contains config key/value pairs.
	Config := ioreader.ConfigLoader(configFilePath)

	//Nodes contains the linux machines' credentials and addresses (Parsed from the csv file) to be able to monitor them.
	//the file name is specified in the config.json file.
	Nodes := ioreader.NodeLoader(filepath.Join("input", Config.InputFileName), Config.SshTunnel)

	//cmds contains the linux bash commands to be executed on the linux machines.
	cmds := []string{

		//CPU Query.
		`cat <(grep 'cpu ' /proc/stat) <(sleep 1 && grep 'cpu ' /proc/stat) | awk -v RS="" '{print ($13-$2+$15-$4)*100/($13-$2+$15-$4+$16-$5)}'`,

		//Disk Query.
		`df -hP`,

		//RAM Query.
		`free -m`,
	}

	//nedata is a channel made from the struct which can contain the linux results per ne.
	//if the query is successful, we expect to receive nedata channel from each node.
	nedata := make(chan retriever.ResourceUtil, len(Nodes))

	//errdata channel will be true in case data query fails from the node.
	errdata := make(chan bool)

	//wg manages the workers to be restricted to the specified value in the config.json file.
	var wg sync.WaitGroup

	//busyWorkers registers how many workers are currently busy to be used for keeping the worker quantity as specified in the config.json file.
	busyWorkers := 0

	//################################################################################################################################################

	//this for{} executes the node iteration based on the configured interval.
	for {

		//here we read and parse the config.json file again so we can change the parameters on the fly and during the execution.
		Config = ioreader.ConfigLoader(configFilePath)

		//qint is the query interval in second.
		qint, _ := strconv.ParseInt(Config.QueryInterval, 10, 16)

		//totalWorkers ==> how many concurrent query we can have.
		totalWorkers, _ := strconv.ParseInt(Config.WorkerQuantity, 10, 64)

		log.Printf("Total Workers: %v\n", totalWorkers)

		//this for{} iterates through the nodes and invoked the retriever.NodeQuery() per node.
		for _, ne := range Nodes {

			//here we control the workers quantity and keep them below/equal the configured value on config.json file.
			//if all x workers are busy, we wait for all of them to finish their job and then continue the loop.
			if int64(busyWorkers) >= totalWorkers {
				log.Println(("Waiting for idle workers..."))
				wg.Wait()
				busyWorkers = 0
			}

			wg.Add(1)
			busyWorkers += 1

			//query the data for this node.
			go retriever.NodeQuery(&busyWorkers, &wg, nedata, errdata, cmds, ne, Config)
		}

		//all nodes are queried. we wait for all remaining workers to return their result.
		wg.Wait()

		//setting all workers to idle state.
		busyWorkers = 0

		//this for collects the result of each node from the nedata channel.
		//if a valid data is collected from the node, it sends the data to UtilizationAssessment() to be parsed.
		//if errdata is true, the query has failed for that node and it will be skipped.
		//################################################################################################################################################
		for _, m := range Nodes {
			select {
			case <-errdata:
				log.Printf("error - no data received from %v\n", m.Name)
				continue
			case result := <-nedata:
				retriever.UtilizationAssessment(Nodes, result)
			}
		}
		//################################################################################################################################################
		//resting as specified in the config file.
		log.Printf("Resting for %v seconds", qint)

		time.Sleep(time.Duration(qint) * time.Second)
	}
}
