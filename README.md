# winc_node_setup V1
Set up windows node and connect to openshift cluster.

## pre-requisite
Have an openshift cluster running on AWS.
configure aws and KUBECONFIG

## What it does
grab vpc name 
create windows container node (win server 2019) under the vpc
properties:
 - name
 - public IP (subnet)
 - security group (secure public IP RDP, Ansible, 10.x/16)

## output
A way to rdp inside of <user>-winc-node

## V2 (future work) Ansible
- filewall
- powershell
