package ec2_instances

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"net/http"
)

func CreateEC2WinC(svc *ec2.EC2, vpcID, imageId, instanceType, keyName string) {

	if imageId == "" {
		imageId = "ami-04ca2d0801450d495" // windows server 2019
	}
	if instanceType == "" {
		instanceType = "t2.micro" // free tier
	}
	if keyName == "" {
		keyName = "libra" // use libra.pem
	}
	// create a subnet based on vpcID
	var subnetID string
	subnet, err := svc.CreateSubnet(&ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.0.0/16"),
		VpcId:     aws.String(vpcID),
	})
	if err != nil {
		//log.Fatalf("Failed to create subnet based on given VpcID: %v, %v", vpcID, err)
		subnets, err := svc.DescribeSubnets(&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("tag-key"),
					Values: aws.StringSlice([]string{"kubernetes.io/role/internal-elb"}), // indication of private net
				},
				//{
				//	Name:   aws.String("cidr-block"),
				//	Values: aws.StringSlice([]string{"10.0.0.0/16"}),
				//},
			},
		})
		if err != nil {
			log.Fatalf("Failed to create or search subnet based on given VpcID: %v, %v", vpcID, err)
		}
		//TODO: exclude private subnets
		subnetID = *subnets.Subnets[0].SubnetId
	} else {
		subnetID = *subnet.Subnet.SubnetId
	}
	sg, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("bsong-winc-node"),
		Description: aws.String("security group for rdp and all traffic"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		log.Fatalf("Could not create Security Group: %v", err)
	}
	// Specify the details of the instance
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String(imageId),
		InstanceType:     aws.String(instanceType),
		KeyName:          aws.String(keyName),
		SubnetId:         aws.String(subnetID),
		MinCount:         aws.Int64(1),
		MaxCount:         aws.Int64(1),
		SecurityGroupIds: []*string{aws.String(*sg.GroupId)},
	})
	if err != nil {
		log.Fatalf("Could not create instance: %v", err)
	} else {
		log.Println("Created instance", *runResult.Instances[0].InstanceId)
	}

	_, err = svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(*sg.GroupId),
		IpPermissions: []*ec2.IpPermission{
			//(&ec2.IpPermission{}).
			//	SetIpProtocol("-1").
			//	SetIpRanges([]*ec2.IpRange{
			//		{CidrIp: aws.String("0.0.0.0/16")},
			//	}),
			(&ec2.IpPermission{}).
				SetIpProtocol("tcp").
				SetFromPort(3389).
				SetToPort(3389).
				SetIpRanges([]*ec2.IpRange{
					(&ec2.IpRange{}).
						SetCidrIp(getMyIp() + "/32"),
				}),
		},
	})
	if err != nil {
		log.Println("Unable to set security group ingress, %v", err)
	}
	// Add tags to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("winc-node"),
			},
		},
	})
	if err != nil {
		log.Println("Could not create tags for instance", runResult.Instances[0].InstanceId, err)
		return
	}

	log.Println("Successfully tagged instance")
}

func getMyIp() string {
	resp, err := http.Get("http://myexternalip.com/raw") //TODO: we need a more reliable strategy
	if err != nil {
		log.Panic("Failed to get external IP Addr")
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		log.Panic("Failed to read external IP Addr")
	}
	return buf.String()
}
