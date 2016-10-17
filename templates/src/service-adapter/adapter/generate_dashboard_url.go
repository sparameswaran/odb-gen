package adapter

import (
	"fmt"
	"reflect"
	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

type DashboardUrlGenerator struct {
}

func (a *DashboardUrlGenerator) DashboardUrl(instanceID string, plan serviceadapter.Plan, manifest bosh.BoshManifest) (serviceadapter.DashboardUrl, error) {
	
	cfDomainRoute := ""
	cfMap := manifest.InstanceGroups[0].Jobs[0].Properties["cf"]
	
    if rec, ok := cfMap.(map[interface{}]interface{}); ok {
        for key, val := range rec {
        	keyStr := key.(string)
            if keyStr == "app_domains" {
            	if (reflect.TypeOf(val).Kind() == reflect.String) {
					cfDomainRoute = val.(string)
				} else if (reflect.TypeOf(val).Kind() == reflect.Slice) {
				    if rec, ok := val.([]interface{}); ok {
				    	cfDomainRoute = rec[0].(string)
				    }														
				}
				break
			} 
        }
    }

	dashboardUrl := fmt.Sprintf("https://{{product['dashboard']}}.%s/", cfDomainRoute)
	return serviceadapter.DashboardUrl{DashboardUrl: dashboardUrl + instanceID}, nil
}
