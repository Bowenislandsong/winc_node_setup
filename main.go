package main

import (
	"github.com/bowenislandsong/winc_node_setup/pkg/config"
	"github.com/bowenislandsong/winc_node_setup/pkg/ec2_instances"
	"log"
	"os"
)

func main() {
	cred_path := os.Args[1]
	svc := config.CredentialConfig(cred_path, "", "")
	vpcID, err := config.GetVPCByInfrastructureName(svc, "bsong-winc-cluster-874xs")
	if err != nil {
		log.Fatalf("We failed to find our vpc %v", err)
	}
	ec2_instances.CreateEC2WinC(svc, vpcID, "", "", "")
}
