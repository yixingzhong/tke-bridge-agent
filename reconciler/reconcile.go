package reconciler

import (
	"bufio"
	"fmt"
	log "github.com/golang/glog"
	"github.com/qyzhaoxun/tke-bridge-agent/cri"
	"io/ioutil"
	"net"
	"os"
	"time"
)

const (
	defaultCheckInterval = 5 * time.Minute
)

type CniReconciler struct {
	allocateInfoPath string
	criClient        cri.CRIAPIs
}

func New(allocateInfoPath string) *CniReconciler {
	if allocateInfoPath == "" {
		allocateInfoPath = "/var/lib/cni/networks/tke-bridge"
	}
	return &CniReconciler{
		allocateInfoPath: allocateInfoPath,
		criClient:        cri.New(),
	}
}

func (cr *CniReconciler) Run(stopCh <-chan struct{}) {
	// check dirty cni at startup
	cr.checkDirtyCNIData()

	ticker := time.NewTicker(defaultCheckInterval)
	for {
		select {
		case <-ticker.C:
			cr.checkDirtyCNIData()
		case <-stopCh:
			return
		}
	}
}

func (cr *CniReconciler) checkDirtyCNIData() {
	log.Infof("start checking if ipam store has dirty cni data in dir: %s==========================>", cr.allocateInfoPath)

	sandboxes, err := cr.criClient.GetReadyPodSandboxes()
	if err != nil {
		log.Errorf("failed to list ready sandboxes, skip checking: %v", err)
		return
	}

	allocInfo, err := cr.getAllocateSet()
	if err != nil {
		log.Errorf("failed to get cni allocated info, skip checking: %v", err)
		return
	}
	log.Infof("get allocated info: %v", allocInfo)

	sandboxesSet := make(map[string]*cri.SandboxInfo)
	for _, sandbox := range sandboxes {
		sandboxesSet[sandbox.ContainerId] = sandbox
	}
	log.Infof("get ready sandboxesSet: %v", sandboxesSet)

	for ip, containerId := range allocInfo {
		if containerId == "" {
			log.Infof("cniReconciler: find ip %s allocated to nothing, delete it from store", ip)
			if err = cr.handleCNIDelete(ip); err != nil {
				// not return, continue to deal with next data
				log.Errorf("cniReconciler: failed to delete dirty ip %v allocated info: %v", ip, err)
			} else {
				log.Infof("cniReconciler: succeed to delete dirty ip %v allocated info", ip)
			}
		}
		if _, ok := sandboxesSet[containerId]; !ok {
			log.Infof("cniReconciler: find ip %s allocated to pod sandbox(%s) not running, delete it from store",
				ip, containerId)
			if err = cr.handleCNIDelete(ip); err != nil {
				// not return, continue to deal with next data
				log.Errorf("cniReconciler: failed to delete dirty ip %v allocated info: %v", ip, err)
			} else {
				log.Infof("cniReconciler: succeed to delete dirty ip %v allocated info", ip)
			}
		}
	}
	log.Infof("check over ===============================================================>")
}

func (cr *CniReconciler) getAllocateSet() (map[string]string, error) {
	res := make(map[string]string)
	dir, err := ioutil.ReadDir(cr.allocateInfoPath)
	if err != nil {
		return nil, err
	}
	for _, fi := range dir {
		if !fi.IsDir() {
			ip := net.ParseIP(fi.Name())
			if ip != nil {
				file, err := os.Open(fmt.Sprintf("%s/%s", cr.allocateInfoPath, fi.Name()))
				if err != nil {
					log.Error("failed to open file %s", fi.Name())
					continue
				}
				// 获取第一行的containerId即可
				scanner := bufio.NewScanner(file)
				var containerId string
				for scanner.Scan() {
					containerId = scanner.Text()
					break
				}
				file.Close()
				res[fi.Name()] = containerId
			}
		}
	}
	return res, nil
}

func (cr *CniReconciler) handleCNIDelete(ip string) error {
	return os.Remove(fmt.Sprintf("%s/%s", cr.allocateInfoPath, ip))
}
