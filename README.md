# winc_node_setup V1
Set up windows node and connect to openshift cluster.

## pre-requisite
Have an openshift cluster running on AWS.
configure aws and KUBECONFIG

## What it does
create windows container node (win server 2019) under the vpc
1. grab openshift Cluster vpc name 
2. Windows Node properties:
    - Node Name \<kerborse\>-winc
    - A m4.large instance
    - Shared vpc with OpenShift Cluster
    - Public Subnet (within the vpc)
    - Auto-assign Public IP
    - Using shared libra key
    - security group (secure public IP RDP with my IP and 10.x/16)
    - Attach IAM role (Openshift Cluster Worker Profile)
    - Attach Security Group (Openshift Cluster - Worker)
Destroy Winows node
1. destroy VM
2. delete security group

## output
A way to rdp inside of <user>-winc-node

## V2 (future work) Ansible
- filewall
- powershell
