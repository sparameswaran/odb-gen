package adapter

import (

	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

func (b *Binder) DeleteBinding(bindingId string, boshVMs bosh.BoshVMs, manifest bosh.BoshManifest, requestParameters serviceadapter.RequestParameters) error {
	/*
	zookeeperServers := boshVMs["zookeeper_server"]
	if len(zookeeperServers) == 0 {
		b.StderrLogger.Println("no VMs for job zookeeper_server")
		return errors.New("")
	}
	*/
	
	// Add any cleanup code here...

	return nil
}
