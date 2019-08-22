# winc_node_setup V1
Set up windows node and connect to openshift cluster.

## pre-requisite
- An existing openshift cluster running on AWS.
- AWS EC2 credentials (aws_access_key_id and aws_access_key_id)
- kubeconfig of OpenShift Cluster

## What it does
create windows container node (win server 2019) under the same vpc as OpenShift Cluster
```bash
winc-setup create
    --vpcid
    --key Default libra
    --Region Default us-east-1
```

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
3. Output a way to rdp inside of Windows node
```bash
winc-setup destroy
```
1. destroy VM
2. delete security group


## V2 (future work) Ansible
- filewall
- powershell
