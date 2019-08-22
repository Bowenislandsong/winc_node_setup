package main

import (
	"github.com/bowenislandsong/winc_node_setup/pkg/config"
	"github.com/bowenislandsong/winc_node_setup/pkg/ec2_instances"
	"log"
)

func main() {
	svc := config.AWSConfig()
	client, err := config.ConfigOpenShift()
	if err != nil {
		log.Fatalf("Failed to get client, %v", err)
	}
	infraID := config.GetInfrastrctureName(client)
	vpcID, err := config.GetVPCByInfrastructureName(svc, infraID)
	if err != nil {
		log.Fatalf("We failed to find our vpc, %v", err)
	}
	ec2_instances.CreateEC2WinC(svc, vpcID, "", "", "")
}
