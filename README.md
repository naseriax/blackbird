# blackbird
Monitor Nokia 1830PSS equipment's RAM/CPU/Disk utilization in Go!

## Current setup:
-   It has the option to connect directly to the node list specified in the nodes.csv file and collect the resource utilization data.
-   It can also use ssh tunnel to connect to a ssh gateway first, and then use the tunnel to reach the nodes in nodes.csv file.
-   Tunnel part is inspired and modified from :https://ixday.github.io/post/golang_ssh_tunneling/ .
-   Choosing between direct connection or tunnel connection can be done via conf/config.json file.
-   So far it only prints the output to the sdtout. I will add logging into a file and email notification later.

![alt text](https://raw.githubusercontent.com/naseriax/token-repo/main/sshtunnel.png)

