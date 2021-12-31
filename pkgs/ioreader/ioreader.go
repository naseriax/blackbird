package ioreader

import (
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type Config struct {
	MailRelayIp   string `json:"mailRelayIp"`
	MailInterval  string `json:"mailInterval"`
	LogfileSize   string `json:"logSize"`
	QueryInterval int    `json:"queryInterval"`
	TotalWorkers  int    `json:"totalWorkers"`
	InputFileName string `json:"InputFileName"`
	SshTunnel     bool   `json:"sshTunnel"`
	SshGwIp       string `json:"sshGwIp"`
	SshGwUser     string `json:"sshGwUser"`
	SshGwPass     string `json:"sshGwPass"`
	SshGwPort     string `json:"sshGwPort"`
}

type Node struct {
	IpAddress        string
	Name             string
	Username         string
	Password         string
	MailNotification string
	CpuThreshold     float64
	RamThreshold     float64
	DiskThreshold    float64
	SshPort          string
	Localport        string
}

func parseCSV(csvdata [][]string) map[string]Node {
	nodes := map[string]Node{}
	floatVals := [3]float64{}

	for _, row := range csvdata[1:] {
		var err error
		for i := range floatVals {
			floatVals[i], err = strconv.ParseFloat(row[i+5], 64)
			if err != nil {
				log.Printf("%v", err.Error())
				floatVals[i] = 99
			}
		}
		tmp := Node{
			IpAddress:        row[0],
			Name:             row[1],
			Username:         row[2],
			Password:         row[3],
			MailNotification: row[4],
			CpuThreshold:     floatVals[0],
			RamThreshold:     floatVals[1],
			DiskThreshold:    floatVals[2],
			SshPort:          row[8],
			Localport:        row[9],
		}
		nodes[tmp.Name] = tmp
	}
	return nodes
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}
	return records
}

func LoadConfig(filePath string) Config {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal("Unable to read the config file "+filePath, err)
	}

	config := Config{}
	_ = json.Unmarshal([]byte(file), &config)

	return config
}

func LoadNode(filename string) map[string]Node {
	records := readCsvFile(filename)
	nodeList := parseCSV(records)
	return nodeList
}
