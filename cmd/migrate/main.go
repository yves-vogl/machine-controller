package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"

	conversions "github.com/kubermatic/machine-controller/pkg/api/conversions"
	downstreammachineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"
	downstreammachines "github.com/kubermatic/machine-controller/pkg/machines"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
	masterURL  string
)

func main() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %v", err)
	}

	apiExtClient := apiextclient.NewForConfigOrDie(cfg)
	clusterv1alpha1Client := clusterv1alpha1clientset.NewForConfigOrDie(cfg)

	if err = migrateIfNecesary(apiExtClient, clusterv1alpha1Client, cfg); err != nil {
		glog.Fatalf("Failed to migrate: %v", err)
	}
}

func migrateIfNecesary(apiextClient apiextclient.Interface,
	clusterv1alpha1Client clusterv1alpha1clientset.Interface,
	config *restclient.Config) error {

	_, err := apiextClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(downstreammachines.CRDName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get crds: %v", err)
	}

	downstreamClient, err := downstreammachineclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create downstream machine client: %v", err)
	}

	return migrateMachines(downstreamClient, clusterv1alpha1Client)
}

func migrateMachines(downstreamClient downstreammachineclientset.Interface,
	clusterv1alpha1Client clusterv1alpha1clientset.Interface) error {

	// Get downstreamMachines
	downstreamMachines, err := downstreamClient.MachineV1alpha1().Machines().List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list downstream machines: %v", err)
	}

	// Convert them all
	var convertedClusterv1alpha1Machines []*clusterv1alpha1.Machine
	for _, downstreamMachine := range downstreamMachines.Items {
		clusterv1alpha1Machine, err := conversions.ConvertV1alpha1DownStreamMachineToV1alpha1ClusterMachine(downstreamMachine)
		if err != nil {
			return fmt.Errorf("failed to convert machine %s: %v", downstreamMachine.Name, err)
		}
		convertedClusterv1alpha1Machines = append(convertedClusterv1alpha1Machines, clusterv1alpha1Machine)
	}

	// Create the new machine, delete the old one, wait for it to be absent
	// We do this in one loop to avoid ending up having all machines in  both the new and the old format if deletion
	// failes for whatever reason
	for _, convertedClusterv1alpha1Machine := range convertedClusterv1alpha1Machines {
		if _, err := clusterv1alpha1Client.ClusterV1alpha1().Machines(convertedClusterv1alpha1Machine.Namespace).Create(convertedClusterv1alpha1Machine); err != nil {
			return fmt.Errorf("failed to create clusterv1alpha1.machine %s: %v", convertedClusterv1alpha1Machine.Name, err)
		}
		//TODO: What about the finalizer?
		if err := downstreamClient.MachineV1alpha1().Machines().Delete(convertedClusterv1alpha1Machine.Name, nil); err != nil {
			return fmt.Errorf("failed to delete machine %s: %v", convertedClusterv1alpha1Machine.Name, err)
		}
		if err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
			return isDownstreamMachineDeleted(convertedClusterv1alpha1Machine.Name, downstreamClient)
		}); err != nil {
			return fmt.Errorf("failed to wait for machine %s to be deleted: %v", convertedClusterv1alpha1Machine.Name, err)
		}
	}
	return nil
}

func isDownstreamMachineDeleted(name string, client downstreammachineclientset.Interface) (bool, error) {
	if _, err := client.MachineV1alpha1().Machines().Get(name, metav1.GetOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
