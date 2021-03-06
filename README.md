# blackbird
Monitor Nokia 1830PSS equipment's RAM/CPU/Disk utilization in Go!

## Current setup:
-   It has the option to connect directly to the node list specified in the nodes.csv file and collect the resource utilization data.
-   It can also use ssh tunnel to connect to a ssh gateway first, and then use the tunnel to reach the nodes in nodes.csv file.
-   Tunnel part is inspired and modified from :https://ixday.github.io/post/golang_ssh_tunneling/ .
-   Choosing between direct connection or tunnel connection can be done via conf/config.json file.
-   So far it only prints the output to the sdtout. I will add logging into a file and email notification later.

![alt text](https://raw.githubusercontent.com/naseriax/token-repo/main/sshtunnel.png)

## config file structure:
```
"mailRelayIp": "1.1.1.1",        <-- SMTP relay server ip address to receive mail notifications(Future use)
"mailInterval" : "1800",         <-- email sending inerval in seconds to avoid mailbox overload (Future use)
"logSize": "10",                 <-- log file rotation triggering size (Future use)
"queryInterval": "20",           <-- time in seconds to wait between each query on nes
"inputFileName": "nodes.csv",    <-- node list file name, it must be located inside input folder
"workerQuantity": "3",           <-- concurrent goroutine quantity. (how many nodes to be connected to, at the same time in parallel)
"sshTunnel" : false,             <-- whether ssh tunnel will be used to connect to the nodes or nodes can be contacted directly
"sshGwIp":"2.2.2.2",             <-- if sshTunnel is true, this ip will be used as ssh server (ssh gateway)
"sshGwUser":"root",              <-- if sshTunnel is true, this username will be used to connect to the ssh server
"sshGwPass" :"pass",             <-- if sshTunnel is true, this password will be used to connect to the ssh server
"sshGwPort":"22"                 <-- if sshTunnel is true, this port will be used to connect to the ssh server (allowed port between local machine anc ssh server)
```

## nodes file structure:
```
ipAddress             <-- ne IP address
name                  <-- ne name, no need to match the actual node name
username              <-- ssh username to be used to connect to the ne
password              <-- ssh password to be used to connect to the ne
mailNotification      <-- whether to send mail notification in case of any utilization violation (Future use)
cpuThreshold          <-- if current cpu utilization value is above this value, it will be logged, mailed and printed
ramThreshold          <-- if current ram utilization value is above this value, it will be logged, mailed and printed
diskThreshold         <-- if current disk utilization values are above this value, it will be logged, mailed and printed
sshPort               <-- allowed ssh port on the ne
localport             <-- if sshTunnel in config file is true, the ssh tunnel will be mapped to this port number on the local machine
```


### Below commands will be executed on the target machine every {interval} seconds:

CPU:
```
cat <(grep 'cpu ' /proc/stat) <(sleep 1 && grep 'cpu ' /proc/stat) | awk -v RS="" '{print ($13-$2+$15-$4)*100/($13-$2+$15-$4+$16-$5)}'
```

RAM:
```
free -m
```

Disk:
```
df -hP
```
