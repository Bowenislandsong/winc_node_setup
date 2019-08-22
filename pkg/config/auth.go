package config

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	v1 "github.com/openshift/api/config/v1"
	client "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
)

func contains(slice *[]interface{}, item *interface{}) bool {
	for i := range *slice {
		if (*slice)[i] == *item {
			return true
		}
	}
	return false
}
func AWSConfig() *ec2.EC2 {
	// Grab settings from flag values
	// TODO: Default values may contain redhat information (consider removing default values before public facing)
	credPath := flag.String("awsconfig", os.Getenv("HOME")+"/.aws/credentials", "Get absolute path of aws credentials")
	credAccount := flag.String("account", "openshift-dev", "Get the account name of the aws credentials") // Default accounts for dev team in OpenShift
	region := flag.String("region", "us-east-1", "Set region where the instance will be running on aws")  // Default region for Boston Office or East Coast (virginia)

	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewSharedCredentials(*credPath, *credAccount),
		Region:      aws.String(*region),
	}))
	svc := ec2.New(sess, aws.NewConfig())
	return svc
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

// get aws instance id


// TODO: consult https://github.com/openshift/cluster-kube-scheduler-operator/blob/master/pkg/operator/configobservation/configobservercontroller/observe_config_controller.go#L49 using informer
// Return openshift Client
func ConfigOpenShift() (*client.Clientset, error) {
	kubeConfig := flag.String("kubeconfig", os.Getenv("HOME")+"/.kube/kubeconfig", "absolute path to the kubeconfig file")
	flag.Parse()
	log.Println("kubeconfig source: ", *kubeConfig)
	c, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read from kubeconfig, %v", err)
	}
	ocClient, err := client.NewForConfig(c)
	if err != nil {
		log.Fatalf("Error conveting rest client into OpenShift versioned client, %v", err)
	}
	thing,_:=ocClient.ConfigV1().Images().List(metav1.ListOptions{})
	println(thing)
	return ocClient, nil
}

func GetInfrastrcture(c *client.Clientset) v1.Infrastructure {

	infra, err := c.ConfigV1().Infrastructures().List(metav1.ListOptions{})
	if err != nil || infra == nil || len(infra.Items) != 1 {
		log.Fatalf("Error getting infrastructure, %v", err)
	}
	return infra.Items[0]
}

//func ConfigOpenShift() (client.Client, error) {
//	c := config.GetConfigOrDie()
//	return client.New(c, client.Options{})
//}
//
//// get infraID
//func GetInfrastrctureName(c client.Client) (string, error) {
//	infra := &configv1.Infrastructure{}
//	err := c.Get(context.Background(), types.NamespacedName{Name: "cluster"}, infra)
//	if err !=nil{
//		print(err.Error())
//	}
//	return infra.Status.InfrastructureName, nil
//	//return utils.LoadInfrastructureName(c, logrus.New())
//}
