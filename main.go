package main

import (
	"github.com/bowenislandsong/winc_node_setup/pkg/config"
	"github.com/bowenislandsong/winc_node_setup/pkg/ec2_instances"
	"log"
)

func main() {
	svc, svcIAM := config.AWSConfig()
	client, err := config.ConfigOpenShift()
	if err != nil {
		log.Fatalf("Failed to get client, %v", err)
	}
	infra := ec2_instances.GetInfrastrcture(client)
	vpcID, err := ec2_instances.GetVPCByInfrastructure(svc, infra)
	if err != nil {
		log.Fatalf("We failed to find our vpc, %v", err)
	}
	ec2_instances.CreateEC2WinC(svc, svcIAM, vpcID, infra.Status.InfrastructureName, "", "", "")
}
