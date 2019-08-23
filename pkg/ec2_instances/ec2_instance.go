package ec2_instances

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	v1 "github.com/openshift/api/config/v1"
	client "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"

	"net/http"
)

func CreateEC2WinC(svc *ec2.EC2, vpcID, infraID, imageId, instanceType, keyName string) {

	if imageId == "" {
		imageId = "ami-0943eb2c39917fc11" // Default using Aravindh's firewall disabled image (Does not always have firewall disabled) AWS windows server 2019 is ami-04ca2d0801450d495
	}
	if instanceType == "" {
		instanceType = "m4.large"
	}
	if keyName == "" {
		keyName = "libra" // use libra.pem
	}
	workerSG := getClusterSGID(svc, infraID, "worker")

	subnetID, err := getPubSubnetOrCreate(svc, vpcID, infraID)
	sg, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(infraID + "winc-sg"),
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
		iam
		SecurityGroupIds: []*string{aws.String(*sg.GroupId), aws.String(workerSG)},
	})
	if err != nil {
		log.Fatalf("Could not create instance: %v", err)
	} else {
		log.Println("Created instance", *runResult.Instances[0].InstanceId)
	}

	_, err = svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(*sg.GroupId),
		IpPermissions: []*ec2.IpPermission{
			(&ec2.IpPermission{}).
				SetIpProtocol("-1").
				SetIpRanges([]*ec2.IpRange{
					(&ec2.IpRange{}).
						SetCidrIp("10.0.0.0/16"),
				}),
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
	log.Println("Successfully created windows node instance")
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

func GetInfrastrcture(c *client.Clientset) v1.Infrastructure {
	infra, err := c.ConfigV1().Infrastructures().List(metav1.ListOptions{})
	if err != nil || infra == nil || len(infra.Items) != 1 { // we should only have 1 infrastructure
		log.Fatalf("Error getting infrastructure, %v", err)
	}
	return infra.Items[0]
}

func GetVPCByInfrastructure(svc *ec2.EC2, infra v1.Infrastructure) (string, error) {
	res, err := svc.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: aws.StringSlice([]string{infra.Status.InfrastructureName + "-vpc"}), //TODO: use this kubernetes.io/cluster/{infraName}: owned
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

func getPubSubnetOrCreate(svc *ec2.EC2, vpcID, infraID string) (string, error) {
	// search subnet by the vpcid
	subnets, err := svc.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{vpcID}), // grab subnet owned by the vpcID
			},
			//{
			//	Name:   aws.String("cidr-block"),
			//	Values: aws.StringSlice([]string{"10.0.0.0/16"}),
			//},
		},
	})
	if err != nil {
		log.Println("failed to search subnet based on given VpcID: %v, %v, will create one instead", vpcID, err)
		// create a subnet based on vpcID
		subnet, err := svc.CreateSubnet(&ec2.CreateSubnetInput{ // create subnet under the vpc (most likely not used since openshift-installer creates 6+ of them)
			CidrBlock: aws.String("10.0.0.0/16"),
			VpcId:     aws.String(vpcID),
		})
		if err != nil {
			log.Fatalf("Failed to search or create public subnet based on given VpcID: %v, %v", vpcID, err)
		}
		return *subnet.Subnet.SubnetId, err
	}
	for _, subnet := range subnets.Subnets { // find public subnet within the vpc
		for _, tag := range subnet.Tags {
			if *tag.Key == "Name" && strings.Contains(*tag.Value, infraID+"-public-") {
				return *subnet.SubnetId, err
			}
		}
	}
	return "", fmt.Errorf("failed to find public subnet in vpc: %v", vpcID)
}

func getClusterSGID(svc *ec2.EC2, infraID, clusterFunction string) string {
	sg, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: aws.StringSlice([]string{infraID + "-" + clusterFunction + "-sg"}),
			},
			{
				Name:   aws.String("tag:kubernetes.io/cluster/" + infraID),
				Values: aws.StringSlice([]string{"owned"}),
			},
		},
	})
	if err != nil {
		log.Panicf("Failed to attach security group of openshift cluster worker, please manually add it, %v", err)
	}
	if sg == nil || len(sg.SecurityGroups) > 1 {
		log.Panicf("nil or more than one security groups are found for the openshift cluster %v nodes, this should not happen, we attached the first one.", clusterFunction)
	}
	return *sg.SecurityGroups[0].GroupId
}

func getIAMrole(svc *ec2.EC2, infraID, clusterFunction string) ec2.IamInstanceProfile {
	iams, err:=svc.DescribeIamInstanceProfileAssociations(&ec2.DescribeIamInstanceProfileAssociationsInput{
		Filters:[]*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: aws.StringSlice([]string{infraID + "-" + clusterFunction + "-role"}),
			},
			{
				Name:   aws.String("tag:kubernetes.io/cluster/" + infraID),
				Values: aws.StringSlice([]string{"owned"}),
			},
		},
	})
	if err!=nil{
		log.Panicf("failed to find iam role, please attache manually")
	}
	return *iams.IamInstanceProfileAssociations[0].IamInstanceProfile
}