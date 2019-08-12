package config

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
)

func Credential_config(cred_path, region, cred_account string) *ec2.EC2 {
	// Default settings
	if cred_account == "" {
		cred_account = "openshift-dev"
	}
	if region == "" {
		region = "us-east-1"
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewSharedCredentials(cred_path, cred_account),
		Region:      aws.String(region),
	}))
	svc := ec2.New(sess, aws.NewConfig())

	list_vpc(svc)
	return svc
}

func list_vpc(svc *ec2.EC2){
	//filter:
	//             "State": "available",
	//
	res, err := svc.DescribeVpcs(nil)
	if err != nil {
		log.Panicf("Unable to describe VPCs, %v", err)
	}
	if len(res.Vpcs) == 0 {
		log.Panicf("No VPCs found to associate security group with.")
	}
	for _, vpc :=range res.Vpcs {
		println(*vpc.VpcId)
	}
}