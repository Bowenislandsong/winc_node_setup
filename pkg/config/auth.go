package config

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	client "github.com/openshift/client-go/config/clientset/versioned"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
)

func AWSConfigSess(credPath, credAccount, region *string) *session.Session {
	// Grab settings from flag values
	// TODO: Default values gear towards RedHat Boston Office (consider removing default values before public facing)
	if _, err := os.Stat(*credPath); os.IsNotExist(err) {
		log.Fatalf("failed to find AWS credentials from path '%v'", *credPath)
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewSharedCredentials(*credPath, *credAccount),
		Region:      aws.String(*region),
	}))
	return sess
}

// Return openshift Client
func OpenShiftConfig(kubeConfigPath *string) (*client.Clientset, error) {
	log.Println("kubeconfig source: ", *kubeConfigPath)
	c, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from path '%v', %v", *kubeConfigPath, err)
	}
	ocClient, err := client.NewForConfig(c)
	if err != nil {
		log.Fatalf("Error conveting rest client into OpenShift versioned client, %v", err)
	}
	return ocClient, nil
}
