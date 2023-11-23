package controller

import (
	"context"
	"fmt"
	cobrautil "hybridctl/util"
	"time"

	namespacefunc "hcp-pkg/kube-resource/namespace"

	"hcp-pkg/util"
	"hcp-pkg/util/clientset"
	"hcp-pkg/util/clusterManager"

	hcpclusterv1alpha1 "hcp-pkg/client/hcpcluster/v1alpha1/clientset/versioned"
	informer "hcp-pkg/client/hcpcluster/v1alpha1/informers/externalversions/hcpcluster/v1alpha1"
	lister "hcp-pkg/client/hcpcluster/v1alpha1/listers/hcpcluster/v1alpha1"
	hcpclusterscheme "hcp-pkg/client/sync/v1alpha1/clientset/versioned/scheme"

	rbacv1 "k8s.io/api/rbac/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	kubefed "sigs.k8s.io/kubefed/pkg/client/generic"
)

var HCP_NAMESPACE string = "hcp"

const controllerAgentName = "hcp-cluster-manager"

const (
	// SuccessSynced is used as part of the Event 'reason' when a Foo is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a Foo fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Foo"
	// MessageResourceSynced is the message used for an Event fired when a Foo
	// is synced successfully
	MessageResourceSynced = "Foo synced successfully"
)

type Controller struct {
	kubeclientset       kubernetes.Interface
	hcpclusterclientset hcpclusterv1alpha1.Interface
	hcpclusterLister    lister.HCPClusterLister
	hcpclusterSynced    cache.InformerSynced
	workqueue           workqueue.RateLimitingInterface
	recorder            record.EventRecorder
}

func NewController(
	kubeclientset kubernetes.Interface,
	hcpclusterclientset hcpclusterv1alpha1.Interface,
	hcpclusterinformer informer.HCPClusterInformer) *Controller {
	utilruntime.Must(hcpclusterscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadCaster := record.NewBroadcaster()
	eventBroadCaster.StartStructuredLogging(0)
	eventBroadCaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(HCP_NAMESPACE)})
	recorder := eventBroadCaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:       kubeclientset,
		hcpclusterclientset: hcpclusterclientset,
		hcpclusterLister:    hcpclusterinformer.Lister(),
		hcpclusterSynced:    hcpclusterinformer.Informer().HasSynced,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "hcpcluster"),
		recorder:            recorder,
	}

	klog.Info("Setting up event handlers")

	hcpclusterinformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enquenehcpcluster,
		UpdateFunc: func(old, new interface{}) {
			controller.enquenehcpcluster(new)
		},
	})

	return controller
}

func (c *Controller) enquenehcpcluster(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
	
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting ClusterManager")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.hcpclusterSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

//

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	hcpcluster, err := c.hcpclusterLister.HCPClusters(namespace).Get(name)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("hcpcluster '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	joinstatus := hcpcluster.Spec.JoinStatus
	clustername := hcpcluster.Name

	var master_config = clientset.MasterConfig
	var hcpcluster_clientset = clientset.HCPClusterClientset

	// JOIN 대기
	if joinstatus == "JOINING" {
		klog.Info("[JOIN START]")
		join_cluster_config, _ := cobrautil.BuildConfigFromFlags(clustername, "/root/.kube/config")
		if JoinCluster(clustername, master_config, join_cluster_config, hcpcluster_clientset) {

			hcpcluster.Spec.JoinStatus = "JOIN"
			_, err = hcpcluster_clientset.HcpV1alpha1().HCPClusters(HCP_NAMESPACE).Update(context.TODO(), hcpcluster, metav1.UpdateOptions{})
			if err != nil {
				klog.Info(err)
				return err
			} else {
				klog.Info("Succeed to join %s", clustername)
			}

		} else {
			klog.Info("Fail to join ", clustername)
		}

	} else if joinstatus == "UNJOINING" {
		klog.Info("[UNJOIN START]")
		join_cluster_config, _ := cobrautil.BuildConfigFromFlags(clustername, "/root/.kube/config")
		if UnJoinCluster(clustername, master_config, join_cluster_config) {
			hcpcluster.Spec.JoinStatus = "UNJOIN"
			_, err = hcpcluster_clientset.HcpV1alpha1().HCPClusters(HCP_NAMESPACE).Update(context.TODO(), hcpcluster, metav1.UpdateOptions{})
			if err != nil {
				klog.Info(err)
				return err
			} else {
				klog.Info("Succeed to unjoin %s", clustername)
			}
		} else {
			klog.Info("Fail to unjoin", clustername)
		}
	} else if joinstatus == "UNREADY" {
		// UNREADY -- JOIN UNSTABLE
		join_cluster_config, _ := cobrautil.BuildConfigFromFlags(clustername, "/root/.kube/config")
		if UnJoinCluster(clustername, master_config, join_cluster_config) {
			if JoinCluster(clustername, master_config, join_cluster_config, hcpcluster_clientset) {
				hcpcluster.Spec.JoinStatus = "JOIN"
				hcpcluster_clientset.HcpV1alpha1().HCPClusters(HCP_NAMESPACE).Update(context.TODO(), hcpcluster, metav1.UpdateOptions{})
				klog.Infof("Succeed to join", clustername)
			} else {
				klog.Infof("Fail to join", clustername)
			}
		} else {
			klog.Infof("Fail to unjoin", clustername)
		}
	} else {
		// JOIN/UNJOIN Cluster 상태 확인
		cm, _ := clusterManager.NewClusterManager()
		cluster_list := cm.Cluster_list
		// kubefedclusterList 존재 여부 확인
		for _, cluster := range cluster_list.Items {

			// check JOIN Cluster
			if joinstatus == "JOIN" {
				if clustername == cluster.Name {
					klog.Infof("%s is in a kubefedclusterList", clustername)
					// kubefedcluster 상태 확인
					if len(cluster.Status.Conditions) > 0 {
						kubefed_Type := cluster.Status.Conditions[0].Type
						if kubefed_Type == "Ready" {
							// JOIN - STABLE
							klog.Infof("%s is in a stable state", clustername)
						} else {
							// JOIN - UNSTABLE -- JOIN /kubefedcluster에 존재 / Ready 상태가 아닌 경우
							klog.Infof("%s is in a unstable state", clustername)
							klog.Info("Type: ", kubefed_Type)
							hcpcluster.Spec.JoinStatus = "UNREADY"
							_, err = hcpcluster_clientset.HcpV1alpha1().HCPClusters(HCP_NAMESPACE).Update(context.TODO(), hcpcluster, metav1.UpdateOptions{})
							if err != nil {
								klog.Info(err)
								return err
							}
						}
					}
				}
				/*
					else {
						// JOIN - UNSTABLE -- JOIN / kubefedcluster에 존재하지 않는 경우
						klog.Infof("%s is in a unstable state", clustername)
						klog.Infof("Try to Join %s again", clustername)
						hcpcluster.Spec.JoinStatus = "UNREADY"
						_, err = hcp_cluster.HcpV1alpha1().HCPClusters(platform).Update(context.TODO(), hcpcluster, metav1.UpdateOptions{})
						if err != nil {
							klog.Info(err)
							return err
						}
					}
				*/
			} else if joinstatus == "UNJOIN" {
				// UNJOIN -- UNJOIN / kubefedcluster에 존재하는 경우
				if clustername == cluster.Name {
					klog.Infof("ERROR: UNJOIN Cluster %s is in a kubefedclusterList", clustername)
					join_cluster_config, _ := cobrautil.BuildConfigFromFlags(clustername, "/root/.kube/config")
					UnJoinCluster(clustername, master_config, join_cluster_config)
				}
			}
		}
	}

	return nil

}

func JoinCluster(clustername string,
	master_config *rest.Config,
	join_cluster_config *rest.Config,
	hcp_cluster *hcpclusterv1alpha1.Clientset) bool {

	master_client := kubernetes.NewForConfigOrDie(master_config)
	join_cluster_client := kubernetes.NewForConfigOrDie(join_cluster_config)

	str, err := util.CmdExec("/root/.vpa/hack/vpa-up.sh " + clustername)
	if err != nil {
		klog.Errorln(err)
	} else {
		klog.Info(str)
	}

	ns := "kube-federation-system"
	// 1. CREATE namespace "kube-federation-system"
	namespace, err_ns := namespacefunc.CreateNamespace(join_cluster_client, ns)
	if err_ns != nil {
		klog.Errorln(err_ns)
		return false
	} else {
		klog.Info("< Step 1 > Create Namespace Resource [" + namespace.Name + "] in " + clustername)
	}

	// 2. CREATE service account
	ServiceAccount := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clustername + "-hcp",
			Namespace: "kube-federation-system",
		},
	}

	sa, err_sa := join_cluster_client.CoreV1().ServiceAccounts("kube-federation-system").Create(context.TODO(), &ServiceAccount, metav1.CreateOptions{})

	if err_sa != nil {
		klog.Errorln(err_sa)
		return false
	} else {
		klog.Info("< Step 2 > Create Namespace Resource [" + sa.Name + "] in " + clustername)
	}

	// 3. CREATE cluster role
	ClusterRole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubefed-controller-manager:" + clustername,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{rbacv1.APIGroupAll},
				Verbs:     []string{rbacv1.VerbAll},
				Resources: []string{rbacv1.ResourceAll},
			},
			{
				NonResourceURLs: []string{rbacv1.NonResourceAll},
				Verbs:           []string{"get"},
			},
		},
	}

	cr, err_cr := join_cluster_client.RbacV1().ClusterRoles().Create(context.TODO(), &ClusterRole, metav1.CreateOptions{})

	if err_cr != nil {
		klog.Errorln(err_cr)
		return false
	} else {
		klog.Info("< Step 3 > Create ClusterRole Resource [" + cr.Name + "] in " + clustername)
	}

	// 4. CREATE Cluster Role Binding
	ClusterRoleBinding := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kubefed-controller-manager:" + ServiceAccount.Name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ClusterRole.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ServiceAccount.Name,
				Namespace: ServiceAccount.Namespace,
			},
		},
	}

	crb, err_crb := join_cluster_client.RbacV1().ClusterRoleBindings().Create(context.TODO(), &ClusterRoleBinding, metav1.CreateOptions{})

	if err_crb != nil {
		klog.Errorln(err_crb)
		return false
	} else {
		klog.Info("< Step 4 > Create ClusterRoleBinding Resource [" + crb.Name + "] in " + clustername)
	}

	time.Sleep(1 * time.Second)

	// 4. GET & CREATE secret (in hcp)
	join_cluster_sa, err_sa1 := join_cluster_client.CoreV1().ServiceAccounts("kube-federation-system").Get(context.TODO(), sa.Name, metav1.GetOptions{})
	if err_sa1 != nil {
		klog.Errorln(err_sa1)
	}
	join_cluster_secret, err_sc := join_cluster_client.CoreV1().Secrets("kube-federation-system").Get(context.TODO(), join_cluster_sa.Secrets[0].Name, metav1.GetOptions{})
	if err_sc != nil {
		klog.Errorln(err_sc)
		return false
	} else {
		klog.Info("< Step 5-1 > Get Secret Resource [" + join_cluster_secret.Name + "] From " + clustername)
	}

	Secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: clustername + "-",
			Namespace:    "kube-federation-system",
		},
		Data: map[string][]byte{
			"token": join_cluster_secret.Data["token"],
		},
	}
	cluster_secret, err_secret := master_client.CoreV1().Secrets("kube-federation-system").Create(context.TODO(), Secret, metav1.CreateOptions{})

	if err_secret != nil {
		klog.Errorln(err_secret)
		return false
	} else {
		klog.Info("< Step 5-2 > Create Secret Resource [" + cluster_secret.Name + "] in " + "master")
	}

	cm, err := clusterManager.NewClusterManager()
	if err != nil {
		fmt.Println(err)
		return false
	}
	var disabledTLSValidations []fedv1b1.TLSValidation

	if cm.Host_config.TLSClientConfig.Insecure {
		disabledTLSValidations = append(disabledTLSValidations, fedv1b1.TLSAll)
	}
	kubefedcluster := &fedv1b1.KubeFedCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "kubefedcluster",
			APIVersion: "core.kubefed.io",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clustername,
			Namespace: "kube-federation-system",
		},
		Spec: fedv1b1.KubeFedClusterSpec{
			APIEndpoint: join_cluster_config.Host,
			CABundle:    join_cluster_secret.Data["ca.crt"],
			SecretRef: fedv1b1.LocalSecretReference{
				Name: cluster_secret.Name,
			},
			DisabledTLSValidations: disabledTLSValidations,
		},
	}

	clientset := kubefed.NewForConfigOrDie(master_config)
	err = clientset.Create(context.TODO(), kubefedcluster)

	if err != nil {
		klog.Errorln(err)
		return false
	} else {
		klog.Info("< Step 6 > Create KubefedCluster Resource [" + clustername + "] in hcp")
	}
	return true
}

func UnJoinCluster(clustername string,
	master_config *rest.Config,
	join_cluster_config *rest.Config) bool {

	master_client2, err := client.New(master_config, client.Options{})
	join_cluster_client := kubernetes.NewForConfigOrDie(join_cluster_config)

	if err != nil {
		klog.Errorln(err)
		return false
	}

	ns := "kube-federation-system"
	/*
		// 1. DELETE namespace "kube-federation-system"
		ns := "kube-federation-system"
		err_ns := join_cluster_client.CoreV1().Namespaces().Delete(context.TODO(), ns, metav1.DeleteOptions{})

		if err_ns != nil {
			klog.Errorln(err_ns)
			return false
		} else {
			klog.Info("< Step 1 > Delete Namespace Resource [" + ns + "] in " + clustername)
		}
	*/

	// 2. DELETE service account
	sa := clustername + "-hcp"
	err_sa := join_cluster_client.CoreV1().ServiceAccounts("kube-federation-system").Delete(context.TODO(), sa, metav1.DeleteOptions{})

	if err_sa != nil {
		klog.Errorln(err_sa)
		return false
	} else {
		klog.Info("< Step 1 > Delete ServiceAccount Resource [" + sa + "] in " + clustername)
	}

	// 3. DELETE cluster role
	cr := "kubefed-controller-manager:" + clustername
	err_cr := join_cluster_client.RbacV1().ClusterRoles().Delete(context.TODO(), cr, metav1.DeleteOptions{})

	if err_cr != nil {
		klog.Errorln(err_cr)
		return false
	} else {
		klog.Info("< Step 2 > Delete ClusterRole Resource [" + cr + "] in " + clustername)
	}

	// 4. DELETE Cluster Role Binding
	crb := "kubefed-controller-manager:" + sa
	err_crb := join_cluster_client.RbacV1().ClusterRoleBindings().Delete(context.TODO(), "kubefed-controller-manager:"+sa, metav1.DeleteOptions{})

	if err_crb != nil {
		klog.Errorln(err_crb)
		return false
	} else {
		klog.Info("< Step 3 >  Delete ClusterRoleBinding Resource [" + crb + "] in " + clustername)
	}

	time.Sleep(1 * time.Second)

	// 5. DELETE Kubefedcluster
	clientset := kubefed.NewForConfigOrDie(master_config)
	kubefedcluster_instance := &fedv1b1.KubeFedCluster{}
	err = clientset.Get(context.TODO(), kubefedcluster_instance, ns, clustername)
	// err = master_client2.Get(context.TODO(), types.NamespacedName{Name: clustername, Namespace: ns}, kubefedcluster_instance)
	if err != nil {
		klog.Errorln(err)
		return false
	}

	err = clientset.Delete(context.TODO(), kubefedcluster_instance, ns, clustername, &client.DeleteOptions{})

	if err != nil {
		klog.Errorln(err)
		return false
	} else {
		klog.Info("< Step 4 >  Delete KubefedCluster Resource [" + clustername + "] in hcp")
	}

	// 6. GET & DELETE secret (in hcp)

	secret_instance := corev1.Secret{}
	// master_client2.Get(context.TODO(), types.NamespacedName{Name: })
	err = master_client2.Get(context.TODO(), types.NamespacedName{Name: kubefedcluster_instance.Spec.SecretRef.Name, Namespace: ns}, &secret_instance)
	if err != nil {
		klog.Errorln(err)
		return false
	}

	err = master_client2.Delete(context.TODO(), &secret_instance, &client.DeleteOptions{})

	if err != nil {
		klog.Errorln(err)
		return false
	} else {
		klog.Info("< Step 5 >  Delete Secret Resource [" + secret_instance.Name + "] in " + "master")
	}

	return true
}
