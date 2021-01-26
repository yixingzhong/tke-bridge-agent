package main

import (
	goflag "flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
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
	ObjectNameField            = "metadata.name"
	Pod_CIDR_Allocation_Result = "tke.cloud.tencent.com/pod-cidrs"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	o := NewOptions()
	cmd := &cobra.Command{
		Use:  "tke-cni-bridge",
		Long: `The tke-cni-bridge is a daemon watch node's pod cidr changes.`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Infof("Config agent options")
			err := o.Config()
			if err != nil {
				log.Fatalf("Failed to config agent options, error %v", err)
			}

			log.Infof("Start tke cni bridge")
			err = ensureBridgeNFCallIptables()
			if err != nil {
				log.Fatal(err)
			}

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

			log.Infof("Run node controller")
			fieldSelector := fields.OneTermEqualSelector(ObjectNameField, nodeName)
			nodeLW := cache.NewListWatchFromClient(client.CoreV1().RESTClient(), "nodes", metav1.NamespaceAll, fieldSelector)
			_, nodeController := cache.NewIndexerInformer(nodeLW, &v1.Node{}, 0, cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					node, ok := obj.(*v1.Node)
					if ok {
						syncPodCidr(node, o)
					}
				},
				UpdateFunc: func(oldObj, newObj interface{}) {
					oldNode, ok1 := oldObj.(*v1.Node)
					newNode, ok2 := newObj.(*v1.Node)
					if ok1 && ok2 && oldNode.Spec.PodCIDR != newNode.Spec.PodCIDR {
						syncPodCidr(newNode, o)
					}
				},
			}, cache.Indexers{})

			stopChan := signals.SetupSignalHandler()

			go nodeController.Run(stopChan)

			if sync := WaitForCacheSync("node", stopChan, nodeController.HasSynced); !sync {
				log.Fatalf("local node cache not sync")
			}

			<-stopChan
		},
	}
	cmd.Flags().AddGoFlagSet(goflag.CommandLine)
	cmd.Flags().Lookup("logtostderr").Value.Set("true")
	o.AddFlags(cmd.Flags())
	cmd.Flags().Parse(os.Args[1:])

	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		log.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	defer log.Flush()
	log.Infof("Start agent ...")

	if err := cmd.Execute(); err != nil {
		log.Fatalf("Failed to start agent, error %v", err)
	}
}

func syncPodCidr(node *v1.Node, o *Options) error {
	log.Infof("Sync pod cidr of node %s into config file", node.Name)
	ipamdAllocationResult, ok := node.Annotations[Pod_CIDR_Allocation_Result]

	if node.Spec.PodCIDR == "" && (!ok || ipamdAllocationResult == "") {
		log.Warningf("node has no pod cidr assigned, skipped")
		return nil
	}
	// 存量节点
	if node.Spec.PodCIDR != "" {
		_, cidr, _ := net.ParseCIDR(node.Spec.PodCIDR)
		err := generateOldBridgeConf(cidr, o.MTU, o.HairpinMode, o.CniConfDir)
		if err != nil {
			log.Errorf("Failed to generate bridge conf : %v", err)
			return err
		}
		if o.AddRule {
			if cidr.IP.IsLoopback() {
				log.Warningf("loopback cidr %+v, skipping add rule", cidr)
			} else {
				err := ensureRule(cidr)
				if err != nil {
					log.Errorf("Failed to ensure rule %+v : %v", cidr, err)
					return err
				}
			}
		}
	} else { // 增量节点
		podCIDRs := make([]*net.IPNet, 0)

		cidrs := strings.Split(ipamdAllocationResult, ",")
		for _, cidr := range cidrs {
			_, podCidr, _ := net.ParseCIDR(cidr)
			podCIDRs = append(podCIDRs, podCidr)
		}

		err := generateNewBridgeConf(podCIDRs, o.MTU, o.HairpinMode, o.CniConfDir)
		if err != nil {
			log.Errorf("Failed to generate bridge conf : %v", err)
			return err
		}
	}
	return nil
}

func ensureBridgeNFCallIptables() error {
	// set net.bridge.bridge-nf-call-iptables=1
	command := exec.Command("modprobe", "br-netfilter")
	out, err := command.CombinedOutput()
	if err != nil {
		log.Warningf("failed to modprobe br-netfilter, error %v, output %s", err, string(out))
	}

	err = ioutil.WriteFile("/proc/sys/net/bridge/bridge-nf-call-iptables", []byte("1"), 0640)
	if err != nil {
		return fmt.Errorf("failed to set bridge-nf-call-iptables, error %v", err)
	}
	return nil
}
