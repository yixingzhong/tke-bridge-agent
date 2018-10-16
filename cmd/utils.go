package main

import (
	"fmt"

	log "github.com/golang/glog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

func WaitForCacheSync(controllerName string, stopCh <-chan struct{}, cacheSyncs ...cache.InformerSynced) bool {
	log.Infof("Waiting for caches to sync for %s controller", controllerName)

	if !cache.WaitForCacheSync(stopCh, cacheSyncs...) {
		utilruntime.HandleError(fmt.Errorf("unable to sync caches for %s controller", controllerName))
		return false
	}

	log.Infof("Caches are synced for %s controller", controllerName)
	return true
}