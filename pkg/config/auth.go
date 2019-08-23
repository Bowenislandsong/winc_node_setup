package config

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	client "github.com/openshift/client-go/config/clientset/versioned"
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
	return ocClient, nil
}
