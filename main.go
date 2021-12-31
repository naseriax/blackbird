/*
blackbird query linux machine's current ram, cpu and disk
utilizations and provides log/mail notification if configured.
it can access the nodes directly from this machine or via a ssh gateway (tunnel).
*/
package main

import (
	"blackbird/pkgs/ioreader"
	"blackbird/pkgs/staff"
	"log"
	"path/filepath"
	"time"
)

type ResultControl struct {
	IsNodeQueryFailed chan bool
	QueryResult       chan ResourceUtil
}

func makeFilePath(rootFolder, cfgFileName string) string {
	return filepath.Join(rootFolder, cfgFileName)
}

func loadConfig(cfgFileName string) ioreader.Config {
	return ioreader.LoadConfig(makeFilePath("conf", cfgFileName))
}

func queryCommands() []string {
	return []string{

		//CPU Query.
		`cat <(grep 'cpu ' /proc/stat) <(sleep 1 && grep 'cpu ' /proc/stat) | awk -v RS="" '{print ($13-$2+$15-$4)*100/($13-$2+$15-$4+$16-$5)}'`,

		//Disk Query.
		`df -hP`,

		//RAM Query.
		`free -m`,
	}
}

func loadNodes(nodesFileName string) map[string]ioreader.Node {
	nodeFilePath := makeFilePath("input", nodesFileName)
	return ioreader.LoadNode(nodeFilePath)
}

func getQueryResults(nodes map[string]ioreader.Node, chs ResultControl) {
	for _, node := range nodes {
		select {
		case <-chs.IsNodeQueryFailed:
			log.Printf("error - no data received from %v\n", node.Name)
			continue
		case result := <-chs.QueryResult:
			AssessUtil(nodes, result)
		}
	}
}

func Sleep(s int) {
	log.Printf("Resting for %v seconds", s)
	time.Sleep(time.Duration(s) * time.Second)
}

func buildHr(workers int) staff.Staff {
	log.Printf("Total Workers: %v\n", workers)
	return staff.Staff{
		BusyWorkers:  0,
		TotalWorkers: workers,
	}
}

func buildResultChan(nodeQty int) ResultControl {
	queryResult := make(chan ResourceUtil, nodeQty)
	isNodeQueryFailed := make(chan bool)
	return ResultControl{
		QueryResult:       queryResult,
		IsNodeQueryFailed: isNodeQueryFailed,
	}
}

func queryAllNodes(nodes map[string]ioreader.Node, hr staff.Staff, cfg ioreader.Config, controlChan ResultControl) {
	for _, node := range nodes {
		hr.WaitIfWorkersBusy()
		hr.AssignWork()

		go QueryFromNode(&hr, controlChan, queryCommands(), node, cfg)
	}
}

func main() {

	cfgFileName := "config2.json"

	nodesFileName := "nodes3.csv"
	nodes := loadNodes(nodesFileName)

	//controlChan contains 2 fields: "QueryResult" which receives the querydata from each node,
	//                               "IsNodeQueryFailed" which receives true if node query fails.
	controlChan := buildResultChan(len(nodes))

	for {

		cfg := loadConfig(cfgFileName)
		hr := buildHr(cfg.TotalWorkers)

		queryAllNodes(nodes, hr, cfg, controlChan)

		hr.LetThemRest()
		getQueryResults(nodes, controlChan)
		Sleep(cfg.QueryInterval)
	}
}
