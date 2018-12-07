package main

import (
	goflag "flag"
	"math/rand"
	"net"
	"os"
	"time"

	log "github.com/golang/glog"
	"github.com/hasura/gitkube/pkg/signals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	ObjectNameField = "metadata.name"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	o := NewOptions()
	cmd := &cobra.Command{
		Use:  "tke-cni-bridge",
		Long: `The tke-cni-bridge is a daemon watch node's pod cidr changes.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Config()
			if err != nil {
				log.Fatalf("Failed to config agent options, error %v", err)
			}
			log.Infof("Start tke cni bridge")
			nodeName := os.Getenv("MY_NODE_NAME")
			if nodeName == "" {
				log.Fatalf("Failed to get node name from env")
			}

			kubeConfig, err := rest.InClusterConfig()
			if err != nil {
				log.Fatalf("Failed to get kube config, error %v", err)
			}

			client, err := kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				log.Fatalf("Failed to new kube client, error %v", err)
			}

			fieldSelector := fields.OneTermEqualSelector(ObjectNameField, nodeName)
			nodeLW := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "nodes", metav1.NamespaceAll, fieldSelector)
			_, nodeController := cache.NewIndexerInformer(nodeLW, &v1.Node{}, 0, cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					node, ok := obj.(*v1.Node)
					if ok {
						syncPodCidr(node.Spec.PodCIDR, o)
					}
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					oldNode, ok1 := oldObj.(*v1.Node)
					newNode, ok2 := newObj.(*v1.Node)
					if ok1 && ok2 && oldNode.Spec.PodCIDR != newNode.Spec.PodCIDR {
						syncPodCidr(newNode.Spec.PodCIDR, o)
					}
				},
			}, cache.Indexers{})

			stopChan := signals.SetupSignalHandler()

			go nodeController.Run(stopChan)

			if sync := WaitForCacheSync("node", stopChan, nodeController.HasSynced); !sync {
				log.Fatalf("local node cache not sync")
			}

			// TODO set net.bridge.bridge-nf-call-iptables=1

			<-stopChan
		},
	}
	o.AddFlags(cmd.Flags())

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Lookup("logtostderr").Value.Set("true")

	goflag.Parse()

	defer log.Flush()
	log.Infof("Start agent ...")

	if err := cmd.Execute(); err != nil {
		log.Fatalf("Failed to start agent, error %v", err)
	}
}

func syncPodCidr(podCidr string, o *Options) error {
	if podCidr == "" {
		log.Warningf("node has no pod cidr assigned, skipped")
		return nil
	}
	_, cidr, err := net.ParseCIDR(podCidr)
	if err != nil {
		return err
	}
	err = generateBridgeConf(cidr, o.MTU, o.HairpinMode)
	if err != nil {
		return err
	}

	return ensureRule(cidr)
}
