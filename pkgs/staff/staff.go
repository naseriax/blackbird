package staff

import (
	"log"
	"sync"
)

type Staff struct {
	WorkerSchedule sync.WaitGroup
	BusyWorkers    int
	TotalWorkers   int
}

func (hr *Staff) LetThemRest() {
	hr.WorkerSchedule.Wait()
	hr.BusyWorkers = 0
}

func (hr *Staff) AssignWork() {
	hr.WorkerSchedule.Add(1)
	hr.BusyWorkers += 1
}

func (hr *Staff) WaitIfWorkersBusy() {
	if hr.BusyWorkers == hr.TotalWorkers {
		log.Println("Waiting for idle workers...")
		hr.LetThemRest()
	}
}
