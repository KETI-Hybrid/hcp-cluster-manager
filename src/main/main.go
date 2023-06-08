package main

import (
	"flag"
	"time"

	controller "hcp-clustermanager/src/controller"
	informers "hcp-pkg/client/hcpcluster/v1alpha1/informers/externalversions"
	"hcp-pkg/util/clusterManager"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/klog"
	"k8s.io/sample-controller/pkg/signals"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	cm, err := clusterManager.NewClusterManager()
	if err != nil {
		klog.Errorln(err)
	}

	stopCh := signals.SetupSignalHandler()
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(cm.Host_kubeClient, time.Second*30)
	hcpclusterInformerFactory := informers.NewSharedInformerFactory(cm.HCPCluster_Client, time.Second*30)
	//
	controller := controller.NewController(cm.Host_kubeClient, cm.HCPCluster_Client, hcpclusterInformerFactory.Hcp().V1alpha1().HCPClusters())
	kubeInformerFactory.Start(stopCh)
	hcpclusterInformerFactory.Start(stopCh)
	if err := controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

}
