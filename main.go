package main

import (
	"goxmon/pkgs/ioreader"
	"goxmon/pkgs/retriever"
	"log"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

func main() {
	configFileName := "config.json"
	configFilePath := filepath.Join("conf", configFileName)
	Config := ioreader.ConfigLoader(configFilePath)
	Nodes := ioreader.NodeLoader(filepath.Join("input", Config.InputFileName), Config.SshTunnel)

	cmds := []string{
		`cat <(grep 'cpu ' /proc/stat) <(sleep 1 && grep 'cpu ' /proc/stat) | awk -v RS="" '{print ($13-$2+$15-$4)*100/($13-$2+$15-$4+$16-$5)}'`, //-- CPU Query
		`df -hP`,  //--------------------------------------------------------------------------------------------------------------------------------- Disk Query
		`free -m`, //--------------------------------------------------------------------------------------------------------------------------------- RAM Query
	}

	ch := make(chan retriever.ResourceUtil, len(Nodes))
	var wg sync.WaitGroup

	busyWorkers := 0

	for {
		Config = ioreader.ConfigLoader(configFilePath)
		qint, _ := strconv.ParseInt(Config.QueryInterval, 16, 16)
		totalWorker, _ := strconv.ParseInt(Config.WorkerQuantity, 10, 64)
		log.Printf("Total Workers: %v\n", totalWorker)

		for _, ne := range Nodes {
			if int64(busyWorkers) >= totalWorker {
				log.Println(("Waiting for idle workers..."))
				wg.Wait()
				busyWorkers = 0
			}
			wg.Add(1)
			busyWorkers += 1
			go retriever.NodeQuery(&busyWorkers, &wg, ch, cmds, ne, Config)
		}
		wg.Wait()
		busyWorkers = 0
		for range Nodes {
			result := <-ch
			retriever.UtilizationAssessment(Nodes, result)

		}
		log.Println("going for sleep...")
		time.Sleep(time.Duration(qint) * time.Second)
	}
}
