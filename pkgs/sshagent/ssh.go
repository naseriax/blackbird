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
	Host     string
	Name     string
	Port     string
	UserName string
	Password string
	Timeout  int
	Client   *ssh.Client
	Session  *ssh.Session
}

func Pipe(ch chan int, writer, reader net.Conn) {
	defer writer.Close()
	defer reader.Close()

	_, err := io.Copy(writer, reader)
	if err != nil {
		log.Printf("failed to copy: %s", err)
	}
	ch <- 1
}

func Tunnel(ch chan int, conn *ssh.Client, local, remote string) {
	lst, err := net.Listen("tcp", local)
	if err != nil {
		if strings.Contains(err.Error(), "bind: address already in use") {
			log.Println("The tunnel is already open, resuing it.")
			return
		}
	}
	here, err := lst.Accept()
	if err != nil {
		panic(err)
	}
	go func(ch chan int, here net.Conn) {
		ch1 := make(chan int)
		there, err := conn.Dial("tcp", remote)
		if err != nil {
			log.Fatalf("failed to dial to remote: %q", err)
		}
		go Pipe(ch1, there, here)
		go Pipe(ch1, here, there)

		<-ch1
		c := <-ch1
		ch <- c
		lst.Close()
	}(ch, here)

}

//Connect connects to the specified server and opens a session (Filling the Client and Session fields in SshAgent struct)
func (s *SshAgent) Connect() error {
	var err error

	config := &ssh.ClientConfig{
		User: s.UserName,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(s.Timeout) * time.Second,
	}
	time.Sleep(time.Second)
	s.Client, err = ssh.Dial("tcp", fmt.Sprintf("%v:%v", s.Host, s.Port), config)
	if err != nil {
		log.Printf("Failed to dial: %v\n", err)
		return err
	}

	return nil
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
	s.Client.Close()
	log.Printf("Closed the ssh session for ne %v.", s.Name)
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
		return sshagent, nil
	}
}
