package main

import (
	"github.com/bowenislandsong/winc_node_setup/pkg/config"
	"github.com/bowenislandsong/winc_node_setup/pkg/ec2_instances"
	"log"
)

func main() {
	sessAWS := config.AWSConfigSess()
	oc, err := config.OpenShiftConfig()
	if err != nil {
		log.Fatalf("Failed to get client, %v", err)
	}
	ec2_instances.CreateEC2WinC(sessAWS, oc, "", "", "")
}
