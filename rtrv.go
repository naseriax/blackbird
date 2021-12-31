package main

import (
	"blackbird/pkgs/ioreader"
	"blackbird/pkgs/sshagent"
	"blackbird/pkgs/staff"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type ResourceUtil struct {
	Cpu  float64
	Ram  float64
	Disk map[string]float64
	Name string
}

func ramParse(res string) float64 {
	memLines := strings.Split(res, "\n")

	var output float64
	for _, l := range memLines {
		if strings.Contains(l, "Mem:") {
			memInfo := strings.Fields(l)
			total, err := strconv.ParseFloat(memInfo[1], 64)
			if err != nil {
				log.Printf("%v", err)
			}
			avail, err := strconv.ParseFloat(memInfo[6], 64)
			if err != nil {
				log.Printf("%v", err)
			}
			output = 100 - ((avail * 100) / total)
			break
		}
	}
	val, err := strconv.ParseFloat(fmt.Sprintf("%.1f", output), 64)
	if err != nil {
		log.Printf("%v", err)
	}
	return val
}

func cpuParse(res string) float64 {
	output, err := strconv.ParseFloat(strings.Replace(res, "\n", "", 1), 64)
	if err != nil {
		log.Printf("%v", err)
	}
	val, err := strconv.ParseFloat(fmt.Sprintf("%.1f", output), 64)
	if err != nil {
		log.Printf("%v", err)
	}
	return val
}

func dskParse(res string) map[string]float64 {
	output := map[string]float64{}
	dskLines := strings.Split(res, "\n")
	for _, l := range dskLines {
		if strings.Contains(l, "/") {
			dskInfo := strings.Fields(l)
			val, err := strconv.ParseFloat(strings.Replace(dskInfo[4], "%", "", 1), 64)
			if err != nil {
				log.Printf("%v", err)
			}
			output[dskInfo[5]] = val
		}
	}

	return output
}

func QueryFromNode(hr *staff.Staff, chs ResultControl, cmds []string, ne ioreader.Node, config ioreader.Config) {
	log.Printf("collecting info from ne %v", ne.Name)
	var err error
	var res string
	sshc := sshagent.SshAgent{}
	result := ResourceUtil{
		Name: ne.Name,
	}
	pipeprogress := make(chan int)
	var ClientConn *ssh.Client

	if config.SshTunnel {
		lstReady := make(chan bool)
		sshConfig := &ssh.ClientConfig{
			User: config.SshGwUser,
			Auth: []ssh.AuthMethod{
				ssh.Password(config.SshGwPass),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Duration(5) * time.Second,
		}

		ClientConn, err = ssh.Dial("tcp", fmt.Sprintf("%v:%v", config.SshGwIp, config.SshGwPort), sshConfig)

		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				log.Println("failed to connect to the ssh gateway - gateway unreachable!")
				hr.WorkerSchedule.Done()
				hr.BusyWorkers -= 1
				chs.IsNodeQueryFailed <- true
				return
			} else {
				log.Fatalf("failed to connect to the ssh server: %q", err)
			}
		}

		go sshagent.Tunnel(pipeprogress, lstReady, ClientConn, fmt.Sprintf("localhost:%v", ne.Localport), fmt.Sprintf("%v:%v", ne.IpAddress, ne.SshPort))

		select {
		case <-lstReady:
			break
		case <-time.After(10 * time.Second):
			log.Printf("failed to create the local listener channel - %v - %v", ne.Name, err)
			hr.WorkerSchedule.Done()
			hr.BusyWorkers -= 1
			chs.IsNodeQueryFailed <- true
			return
		}

		sshc, err = sshagent.Init(ne.Name, "localhost", ne.Localport, ne.Username, ne.Password, 10)
		if err != nil {
			log.Printf("connect() error - %v - %v", ne.Name, err)
			ClientConn.Close()
			hr.WorkerSchedule.Done()
			hr.BusyWorkers -= 1
			chs.IsNodeQueryFailed <- true
			return
		}

	} else {
		sshc, err = sshagent.Init(ne.Name, ne.IpAddress, ne.SshPort, ne.Username, ne.Password, 5)
		if err != nil {
			log.Printf("connection error - %v - %v", ne.Name, err)
			ClientConn.Close()
			hr.WorkerSchedule.Done()
			hr.BusyWorkers -= 1
			chs.IsNodeQueryFailed <- true
			return
		}
	}

	for _, c := range cmds {
		res, err = sshc.Exec(c)
		if err != nil {
			sshc.Disconnect()
			log.Printf("command exec error - %v - %v", ne.Name, err)
			ClientConn.Close()
			hr.WorkerSchedule.Done()
			chs.IsNodeQueryFailed <- true
			return
		}
		if strings.Contains(c, "awk") {
			result.Cpu = cpuParse(res)
		} else if strings.Contains(c, "free -m") {
			result.Ram = ramParse(res)
		} else if strings.Contains(c, "df -h") {
			result.Disk = dskParse(res)
		}
	}
	sshc.Disconnect()

	func() {
		for {
			select {
			case <-pipeprogress:
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()
	ClientConn.Close()
	chs.QueryResult <- result
	hr.WorkerSchedule.Done()
	hr.BusyWorkers -= 1
}

func AssessUtil(nodes map[string]ioreader.Node, result ResourceUtil) {
	if result.Cpu >= nodes[result.Name].CpuThreshold {
		log.Printf("Crossed the cpu threshold in NE: %v - Current Value: %v, Threshold: %v\n", result.Name, result.Cpu, nodes[result.Name].CpuThreshold)
	}
	if result.Ram >= nodes[result.Name].RamThreshold {
		log.Printf("Crossed the ram  threshold in NE: %v - Current Value: %v, Threshold: %v\n", result.Name, result.Ram, nodes[result.Name].RamThreshold)
	}

	for mp, val := range result.Disk {
		if val >= nodes[result.Name].DiskThreshold {
			log.Printf("Crossed the disk threshold in NE: %v - for mountpoint %v- Current Value: %v, Threshold: %v\n", result.Name, mp, val, nodes[result.Name].DiskThreshold)
		}
	}
}
