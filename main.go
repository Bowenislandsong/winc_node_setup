package main

import (
	"github.com/bowenislandsong/winc_node_setup/pkg/config"
	"github.com/bowenislandsong/winc_node_setup/pkg/ec2_instances"
	"log"
	"os"
)

func main() {
	aws_cred_path := os.Args[1]
	infraID, _ := config.GetInfraID()
	svc := config.CredentialConfig(aws_cred_path, "", "")
	vpcID, err := config.GetVPCByInfrastructureName(svc, infraID)
	if err != nil {
		log.Fatalf("We failed to find our vpc %v", err)
	}
	ec2_instances.CreateEC2WinC(svc, vpcID, "", "", "")
}
