package k8sMetadataUtil

import (
	"context"
	"fmt"
	"os"
	"time"

	cliClient "github.com/datreeio/admission-webhook-datree/pkg/clients"
	"github.com/datreeio/admission-webhook-datree/pkg/enums"
	"github.com/datreeio/datree/pkg/deploymentConfig"
	"github.com/datreeio/datree/pkg/networkValidator"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func InitK8sMetadataUtil() {

	validator := networkValidator.NewNetworkValidator()
	cliClient := cliClient.NewCliServiceClient(deploymentConfig.URL, validator)

	var clusterUuid k8sTypes.UID
	var nodesCount *int
	var nodesCountErr error

	k8sClient, err := getClientSet()

	if err != nil {
		sendK8sMetadata(nodesCount, nodesCountErr, clusterUuid, cliClient)
		return
	}

	clusterUuid, _ = getClusterUuid(k8sClient)
	nodesCount, nodesCountErr = getNodesCount(k8sClient)

	cornJob := cron.New(cron.WithLocation(time.UTC))
	cornJob.AddFunc("@every 10s", func() { sendK8sMetadata(nodesCount, nodesCountErr, clusterUuid, cliClient) })
	// cornJob.AddFunc("@hourly", func() { sendK8sMetadata(nodesCount, nodesCountErr, clusterUuid, cliClient) })
	cornJob.Start()
}

func getNodesCount(clientset *kubernetes.Clientset) (*int, error) {
	var nodesCount *int

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error getting nodes count", err)
		return nodesCount, err
	}

	totalNodes := len(nodes.Items)
	nodesCount = &totalNodes

	return nodesCount, nil
}

func getClientSet() (*kubernetes.Clientset, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func getClusterUuid(clientset *kubernetes.Clientset) (k8sTypes.UID, error) {
	clusterMetadata, err := clientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return clusterMetadata.UID, nil
}

func sendK8sMetadata(nodesCount *int, nodesCountErr error, clusterUuid k8sTypes.UID, client *cliClient.CliClient) {
	token := os.Getenv(enums.Token)

	var nodesCountErrString string
	if nodesCountErr != nil {
		nodesCountErrString = nodesCountErr.Error()
	}

	client.ReportK8sMetadata(&cliClient.ReportK8sMetadataRequest{
		ClusterUuid:   clusterUuid,
		Token:         token,
		NodesCount:    nodesCount,
		NodesCountErr: nodesCountErrString,
	})
}
