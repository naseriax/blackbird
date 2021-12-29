package sshagent

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

//SshAgent object contains all ssh connectivity info and tools.
type SshAgent struct {
	Host         string
	Name         string
	Port         string
	UserName     string
	Password     string
	Timeout      int
	Client       *ssh.Client
	Session      *ssh.Session
	ClientStatus bool
}

func Pipe(copyProgress chan int, writer, reader net.Conn) {
	defer writer.Close()
	defer reader.Close()

	_, err := io.Copy(writer, reader)
	if err != nil {
		log.Printf("failed to copy: %s", err)
	}
	copyProgress <- 1
}

func Tunnel(pipeProgress chan int, lstReady chan bool, conn *ssh.Client, local, remote string) {
	lst, err := net.Listen("tcp", local)
	if err != nil {
		if strings.Contains(err.Error(), "bind: address already in use") {
			log.Println("The tunnel is already open, resuing it.")
			return
		}
	}

	lstReady <- true

	here, err := lst.Accept()
	if err != nil {
		panic(err)
	}
	go func(pipeProgress chan int, here net.Conn) {
		copyProgress := make(chan int)
		there, err := conn.Dial("tcp", remote)
		if err != nil {
			log.Printf("failed to dial to remote: %q", err)
			lst.Close()
			return
		}
		go Pipe(copyProgress, there, here)
		go Pipe(copyProgress, here, there)
		<-copyProgress
		<-copyProgress
		pipeProgress <- 1
		lst.Close()
	}(pipeProgress, here)

}

//Connect connects to the specified server and opens a session (Filling the Client and Session fields in SshAgent struct)
func (s *SshAgent) Connect() error {
	neReady := make(chan bool)
	var err error

	config := &ssh.ClientConfig{
		User: s.UserName,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(s.Timeout) * time.Second,
	}

	go func(neReady chan bool) {
		s.Client, err = ssh.Dial("tcp", fmt.Sprintf("%v:%v", s.Host, s.Port), config)
		if err != nil {
			log.Printf("Failed to dial: %v\n", err)
			time.Sleep(11 * time.Second)
		}
		neReady <- true
	}(neReady)

	select {
	case <-neReady:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("failed to connect to remote node")
	}

}

//Exec executed a single command on the ssh session.
func (s *SshAgent) Exec(cmd string) (string, error) {
	var err error
	s.Session, err = s.Client.NewSession()
	if err != nil {
		log.Printf("Failed to create session: %v\n", err)
	}
	var b bytes.Buffer
	s.Session.Stdout = &b
	if err := s.Session.Run(cmd); err != nil {
		s.Session.Close()
		return "", fmt.Errorf("failed to run: %v >> %v", cmd, err.Error())
	} else {
		s.Session.Close()
		return b.String(), nil
	}
}

//Disconnect closes the ssh sessoin and connection.
func (s *SshAgent) Disconnect() {
	if s.ClientStatus {
		s.Client.Close()
		s.ClientStatus = false
	}
	s.Client.Close()
}

//Init initialises the ssh connection and returns the usable ssh agent.
func Init(name, host, port, username, password string, timeout int) (SshAgent, error) {
	sshagent := SshAgent{
		Host:     host,
		Port:     port,
		Name:     name,
		UserName: username,
		Password: password,
		Timeout:  timeout,
	}

	err := sshagent.Connect()

	if err != nil {
		return sshagent, err
	} else {
		sshagent.ClientStatus = true
		return sshagent, nil
	}
}
