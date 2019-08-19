package config

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	apiconfigv1 "github.com/openshift/api/config/v1"
	configver "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	occonfig "github.com/openshift/client-go/config/listers/config/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	_ "k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/exec"
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

// get aws instance id

// Bash way of extracting infraID
// TODO: find out how to use LoadInfrastructureName (problem is to match the versions of library)
func GetInfraID() (string, error) {
	out, err := exec.Command("bash", "-c", "oc get infrastructure -o yaml | grep infrastructureName").Output()
	if err != nil || len(out) == 0 {
		return "", err
	}
	return string(out[24 : len(out)-1]), nil
}

// TODO: consult https://github.com/openshift/cluster-kube-scheduler-operator/blob/master/pkg/operator/configobservation/configobservercontroller/observe_config_controller.go#L49 using informer
// create openshift Client
func ConfigOpenShift() (*restclient.Config, error) {
	kubeconfig := flag.String("kubeconfig", os.Getenv("HOME")+"/.kube/kubeconfig", "absolute path to the kubeconfig file")
	flag.Parse()
	println(*kubeconfig)
	c, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read from kubeconfig, %v", err)
	}
	return c, nil
}

func GetInfrastrctureName(c *restclient.Config)string{
	client,err := configver.NewForConfig(c)
	if err !=nil{
		log.Fatalf("Error conveting rest client into versioned client, %v",err)
	}
	configinformer:=configinformers.NewSharedInformerFactory(client, 0).Config().V1().Infrastructures().Lister()
	res, err:= configinformer.Get("Infrastructure")
	if err !=nil{
		log.Fatalf("Error getting stuff, %v",err)
	}
	println(res.Status.InfrastructureName)
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := indexer.Add(&apiconfigv1.Infrastructure{ObjectMeta: v1.ObjectMeta{Name: "cluster"}, }); err != nil {
		log.Fatal(err.Error())
	}

	a, err := occonfig.NewInfrastructureLister(indexer).Get("infraid")
	println(err.Error())
	print(a)
	return ""
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
