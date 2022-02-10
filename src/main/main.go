package main

import (
	controller "Hybrid_Cluster/hcp-cluster-manager/src/controller"
	hcpclusterv1alpha1 "Hybrid_Cluster/pkg/client/hcpcluster/v1alpha1/clientset/versioned"
	informers "Hybrid_Cluster/pkg/client/hcpcluster/v1alpha1/informers/externalversions"
	"Hybrid_Cluster/util/clusterManager"
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/klog/v2"
	"k8s.io/sample-controller/pkg/signals"
)

func main() {

	klog.InitFlags(nil)
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	cm := clusterManager.NewClusterManager()
	hcpcluster_client, err := hcpclusterv1alpha1.NewForConfig(cm.Host_config)
	if err != nil {
		klog.Info(err)
	}
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(cm.Host_kubeClient, time.Second*30)
	hcpclusterInformerFactory := informers.NewSharedInformerFactory(hcpcluster_client, time.Second*30)
	//
	controller := controller.NewController(cm.Host_kubeClient, hcpcluster_client, hcpclusterInformerFactory.Hcp().V1alpha1().HCPClusters())
	kubeInformerFactory.Start(stopCh)
	hcpclusterInformerFactory.Start(stopCh)
	if err := controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

}
