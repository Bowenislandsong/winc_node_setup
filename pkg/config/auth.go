package config

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/openshift/cloud-credential-operator/pkg/controller/utils"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func CredentialConfig(cred_path, region, cred_account string) *ec2.EC2 {
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
	return svc
}

func GetVPCByInfrastructureName(svc *ec2.EC2, infrastructureName string) (string, error) {
	res, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: aws.StringSlice([]string{infrastructureName + "-vpc"}),
			},
			{
				Name:   aws.String("state"),
				Values: aws.StringSlice([]string{"available"}),
			},
		},
	})
	if err != nil {
		log.Panicf("Unable to describe VPCs, %v", err)
	}
	if len(res.Vpcs) == 0 {
		log.Panicf("No VPCs found.")
		return "", err
	} else if len(res.Vpcs) > 1 {
		log.Panicf("More than one VPCs are found, we returned the first one")
	}
	return *res.Vpcs[0].VpcId, err
}

// create openshift Client
func ConfigOpenShift() (client.Client, error) {
	c := config.GetConfigOrDie()
	return client.New(c, client.Options{})
}

// get infraID
func GetInfrastrctureName(c client.Client) (string, error) {
	return utils.LoadInfrastructureName(c, nil)
}
