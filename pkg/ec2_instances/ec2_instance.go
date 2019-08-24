package ec2_instances

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	v1 "github.com/openshift/api/config/v1"
	client "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"

	"net/http"
)

func CreateEC2WinC(sess *session.Session, clientset *client.Clientset, imageId, instanceType, keyName string) {
	svc := ec2.New(sess, aws.NewConfig())
	svcIAM := iam.New(sess, aws.NewConfig())
	var sgID, instanceID *string
	if imageId == "" {
		imageId = "ami-04ca2d0801450d495" // Default using Aravindh's firewall disabled image ami-0943eb2c39917fc11 (Does not always have firewall disabled) AWS windows server 2019 is ami-04ca2d0801450d495
	}
	if instanceType == "" {
		instanceType = "m4.large"
	}
	if keyName == "" {
		keyName = "libra" // use libra.pem
	}
	// get infrastructure from OC using kubeconfig info
	infra := getInfrastrcture(clientset)
	// get infraID an unique readable id for the infrastructure
	infraID := infra.Status.InfrastructureName
	// get vpc id of the openshift cluster
	vpcID, err := getVPCByInfrastructure(svc, infra)
	if err != nil {
		log.Fatalf("We failed to find our vpc, %v", err)
	}
	// get openshift cluster worker security groupID
	workerSG := getClusterSGID(svc, infraID, "worker")
	// get openshift cluster worker iam profile
	iamprofile := getIAMrole(svcIAM, infraID, "worker") // unnecessary, could just rely on naming convention to set the iam specifics
	// get or create a public subnet under the vpcID
	subnetID, err := getPubSubnetOrCreate(svc, vpcID, infraID)
	sg, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(infraID + "-winc-sg"),
		Description: aws.String("security group for rdp and all traffic"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		log.Printf("could not create Security Group, attaching existing instead: %v", err)
		sgs, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcID}),
				},
				{
					Name:   aws.String("group-name"),
					Values: aws.StringSlice([]string{infraID + "-winc-sg"}),
				},
			},
		})
		if err != nil || sgs == nil || len(sgs.SecurityGroups) == 0 {
			log.Fatalf("failed to crate or find security group, %v", err)
		}
		sgID = sgs.SecurityGroups[0].GroupId
	} else {
		sgID = sg.GroupId
	}
	// Specify the details of the instance
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:            aws.String(imageId),
		InstanceType:       aws.String(instanceType),
		KeyName:            aws.String(keyName),
		SubnetId:           aws.String(subnetID),
		MinCount:           aws.Int64(1),
		MaxCount:           aws.Int64(1),
		IamInstanceProfile: iamprofile,

		SecurityGroupIds: []*string{sgID, aws.String(workerSG)},
	})
	if err != nil {
		log.Fatalf("Could not create instance: %v", err)
	} else {
		instanceID = runResult.Instances[0].InstanceId
		log.Println("Created instance", *instanceID)
	}
	ipID := allocatePublicIp(svc)
	err = svc.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{instanceID},
	})
	if err != nil {
		log.Printf("failed to wait for instance to be ok, %v", err)
	}
	_, err = svc.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: ipID,
		InstanceId:   instanceID,
	})
	if err != nil {
		log.Printf("failed to associate public ip for instance, %v", err)
	}
	_, err = svc.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: sgID,
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
		log.Printf("unable to set security group ingress, %v", err)
	}
	// Add tags to the created instance
	_, err = svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(infraID + "-winNode"),
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

func allocatePublicIp(svc *ec2.EC2) *string {
	ip, err := svc.AllocateAddress(&ec2.AllocateAddressInput{})
	if err != nil {
		log.Printf("failed to allocate an elastic ip, please assign public ip manually, %v", err)
	}
	return ip.AllocationId
}

func getInfrastrcture(c *client.Clientset) v1.Infrastructure {
	infra, err := c.ConfigV1().Infrastructures().List(metav1.ListOptions{})
	if err != nil || infra == nil || len(infra.Items) != 1 { // we should only have 1 infrastructure
		log.Fatalf("Error getting infrastructure, %v", err)
	}
	return infra.Items[0]
}

func getVPCByInfrastructure(svc *ec2.EC2, infra v1.Infrastructure) (string, error) {
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
	//vpcAttri, err := svc.DescribeVpcAttribute(&ec2.DescribeVpcAttributeInput{
	//	Attribute:aws.String(ec2.VpcAttributeNameEnableDnsSupport),
	//	VpcId: res.Vpcs[0].VpcId,
	//})
	//if err != nil {
	//	log.Printf("failed to find vpc attribute, no public DNS assigned, %v", err)
	//}
	//vpcAttri.SetEnableDnsHostnames(&ec2.AttributeBooleanValue{Value: aws.Bool(true)})
	//vpcAttri.SetEnableDnsSupport(&ec2.AttributeBooleanValue{Value: aws.Bool(true)})
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
		},
	})
	if err != nil {
		log.Printf("failed to search subnet based on given VpcID: %v, %v, will create one instead", vpcID, err)
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

func getIAMrole(svcIAM *iam.IAM, infraID, clusterFunction string) *ec2.IamInstanceProfileSpecification {
	iamspc, err := svcIAM.GetInstanceProfile(&iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(fmt.Sprintf("%s-%s-profile", infraID, clusterFunction)),
	})
	if err != nil {
		log.Panicf("failed to find iam role, please attache manually %v", err)
	}
	return &ec2.IamInstanceProfileSpecification{
		Arn: iamspc.InstanceProfile.Arn,
	}
}
