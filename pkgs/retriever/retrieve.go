package retriever

import (
	"fmt"
	"goxmon/pkgs/ioreader"
	"goxmon/pkgs/sshagent"
	"log"
	"strconv"
	"strings"
	"sync"
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

func NodeQuery(busyWorkers *int, wg *sync.WaitGroup, ch chan ResourceUtil, cmds []string, ne ioreader.Node, config ioreader.Config) {
	log.Printf("collecting info from ne %v", ne.Name)
	var err error
	sshc := sshagent.SshAgent{}
	result := ResourceUtil{
		Name: ne.Name,
	}
	chx := make(chan int)
	var ClientConn *ssh.Client

	if config.SshTunnel {
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
			log.Fatalf("failed to connect to the ssh server: %q", err)
		}

		go sshagent.Tunnel(chx, ClientConn, fmt.Sprintf("localhost:%v", ne.Localport), fmt.Sprintf("%v:%v", ne.IpAddress, ne.SshPort))

		log.Printf("Created the ssh tunnel for %v (%v) - Local interface: localhost:%v.\n", ne.IpAddress, ne.Name, ne.Localport)

		sshc, err = sshagent.Init(ne.Name, "localhost", ne.Localport, ne.Username, ne.Password, 10)
		if err != nil {
			log.Printf("connection error - %v - %v", ne.Name, err)
		}

	} else {
		sshc, err = sshagent.Init(ne.Name, ne.IpAddress, ne.SshPort, ne.Username, ne.Password, 10)
	}

	if err != nil {
		log.Printf("connection error - %v - %v", ne.Name, err)
	}

	for _, c := range cmds {
		res, _ := sshc.Exec(c)
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
			case <-chx:
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()

	ClientConn.Close()
	ch <- result
	wg.Done()
	*busyWorkers -= 1
}

func UtilizationAssessment(nodes map[string]ioreader.Node, result ResourceUtil) {
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
