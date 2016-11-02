package adapter

import (
	"errors"
	"fmt"
	"strings"	
	//"log"
	//"os"
	"reflect"
	"time"
	"math/rand"
	"strconv"
	"gopkg.in/yaml.v2"
	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

const OnlyStemcellAlias = "only-stemcell"
const PARENT_NODE_FOR_ADDN_ATTRS = "parent_node_for_attributes"

func init() {
    rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandStringRunes(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}

// Map the Instance group of set of jobs running within the vm
func defaultDeploymentInstanceGroupsToJobs() map[string][]string {
	return map[string][]string{
		{% for jobInstance in vmInstances %}
		"{{jobInstance.name}}":     []string{ {{ jobInstance['job_types_as_str'] }}	},
		{% endfor %}
		// "test-errand": []string{"test-errand"},		
		// Retrieve properties and plug in cf propertiers...
	}
}

func (a *ManifestGenerator) GenerateManifest(serviceDeployment serviceadapter.ServiceDeployment,
	servicePlan serviceadapter.Plan,
	requestParams serviceadapter.RequestParameters,
	previousManifest *bosh.BoshManifest,
	previousPlan *serviceadapter.Plan,
) (bosh.BoshManifest, error) {

	// If this is an update, handle the changes inside updateManifest and return the manifest
	if (previousManifest != nil) {
		return UpdateManifest(serviceDeployment, servicePlan, requestParams, previousManifest, previousPlan)
	}

	//logger := log.New(os.Stderr, "[{{product['name']}}] ", log.LstdFlags)
	var releases []bosh.Release

	for _, serviceRelease := range serviceDeployment.Releases {
		releases = append(releases, bosh.Release{
			Name:    serviceRelease.Name,
			Version: serviceRelease.Version,
		})
	}
	
	servicePlanType := servicePlan.Properties["type"]

	a.StderrLogger.Printf("Service Releases: %+v\n", releases)
	a.StderrLogger.Printf("Service Plan Type: %s\n", servicePlanType)

	deploymentInstanceGroupsToJobs := defaultDeploymentInstanceGroupsToJobs()

	err := checkInstanceGroupsPresent([]string{
												{% for jobInstance in vmInstances %}
												"{{jobInstance.name}}",
												{% endfor %}
												// "test-errand"
												}, servicePlan.InstanceGroups)
	if err != nil {
		a.StderrLogger.Println(err.Error())
		return bosh.BoshManifest{}, errors.New("Contact your operator, service configuration issue occurred")
	}

	instanceGroups, err := InstanceGroupMapper(servicePlan.InstanceGroups, serviceDeployment.Releases, OnlyStemcellAlias, deploymentInstanceGroupsToJobs)
	if err != nil {
		a.StderrLogger.Println(err.Error())
		return bosh.BoshManifest{}, errors.New("")
	}

	arbitraryParameters := requestParams.ArbitraryParams()
	a.StderrLogger.Printf("%+v", arbitraryParameters)
	
	deploymentInstanceId := strings.Replace(serviceDeployment.DeploymentName, "service-instance_", "", 1)
		
	{% for jobInstance in vmInstances %}
    {% for jobType in jobInstance['job_types'] %}
    
	{{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user := RandStringRunes(12)
	{{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd := RandStringRunes(20)
	
	{% endfor %}
	
	{{jobInstance['nameInGo']}}Route := arbitraryParameters["{{jobInstance['nameInGo']}}_route"]
	if ({{jobInstance.nameInGo}}Route == nil) {
		{{jobInstance.nameInGo}}Route = fmt.Sprintf("{{jobInstance['name']}}-%s", deploymentInstanceId)
	}

	{% endfor %}

	/*
		A Job instance can have job properties:
		some values can come from Plan properties and others from the reequest json payload
		If all the additional properties are under one node: 'parent_node_for_attributes'
		resulting job properties would be:
		properties:
		  parent_node_for_attributes:
		    attribute1: someValue1,...
		    attribute2: someValue2,...
	*/
	sampleAttributeMap := map[string]interface{}{
		"placeHolder": "testvalue",
	}

	attribute1 := arbitraryParameters["attribute1"]
	attribute2 := arbitraryParameters["attribute2"]
	if (attribute1 != nil) {
		sampleAttributeMap["attribute1"] =  attribute1
	}
	if (attribute2 != nil) {
		sampleAttributeMap["attribute2"] =  attribute2
	}


	{% for jobInstance in vmInstances %}
	{{jobInstance.nameInGo}}InstanceGroup := &instanceGroups[{{jobInstance.index}}]

	if len({{jobInstance.nameInGo}}InstanceGroup.Networks) != 1 {
		a.StderrLogger.Println(fmt.Sprintf("expected 1 network for %s, got %d", {{jobInstance.nameInGo}}InstanceGroup.Name, len({{jobInstance.nameInGo}}InstanceGroup.Networks)))
		return bosh.BoshManifest{}, errors.New("")
	}

	{{jobInstance.nameInGo}}InstanceParams := arbitraryParameters["{{jobInstance.nameInGo}}_instances"]
	if ({{jobInstance.nameInGo}}InstanceParams != nil) {
		if floatval64, ok := {{jobInstance.nameInGo}}InstanceParams.(float64); ok {
		    {{jobInstance.nameInGo}}InstanceGroup.Instances = int(floatval64)
		} else if intval, ok := {{jobInstance.nameInGo}}InstanceParams.(int); ok {
		    {{jobInstance.nameInGo}}InstanceGroup.Instances = int(intval)
		} else if str, ok := {{jobInstance.nameInGo}}InstanceParams.(string); ok {
			val, _ := strconv.ParseInt(str,10, 0)
			{{jobInstance.nameInGo}}InstanceGroup.Instances = int(val)
		}
	}

	/*
	{% for jobType in jobInstance['job_types'] %}
	if {{jobInstance.nameInGo}}_{{jobType['nameInGo']}}_Job, ok := getJobFromInstanceGroup("{{jobType['name']}}", {{jobInstance.nameInGo}}InstanceGroup); ok {
		{{jobInstance.nameInGo}}_{{jobType['nameInGo']}}_Job.Properties = map[string]interface{}{
			"network": {{jobInstance.nameInGo}}InstanceGroup.Networks[0].Name,
			"address": {{jobInstance.nameInGo}}Route,
			"{{jobType['nameInGo']}}_username": {{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user,
			"{{jobType['nameInGo']}}_password": {{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd,
			// Add other instance creds
			{% for otherJobInstance in vmInstances %}

			{% if otherJobInstance != jobInstance %}
			"{{otherJobInstance['nameInGo']}}_address": {{otherJobInstance['nameInGo']}}Route,
			{% for jobType in otherJobInstance['job_types'] %}
			"{{otherJobInstance.nameInGo}}_{{jobType['nameInGo']}}_username": {{otherJobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user ,
			"{{otherJobInstance.nameInGo}}_{{jobType['nameInGo']}}_password": {{otherJobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd,
			{% endfor %}
			{% endif %}
			{% endfor %}
		}
		for key, val := range servicePlan.Properties {
    		{{jobInstance.nameInGo}}_{{jobType['nameInGo']}}_Job.Properties[key] = val
    	}
	}
	{% endfor %}
	
	*/

	{{jobInstance['nameInGo']}}InstanceGroup.Properties = map[string]interface{}{
		"network": {{jobInstance['nameInGo']}}InstanceGroup.Networks[0].Name,
		"address": {{jobInstance['nameInGo']}}Route,
		{% for jobType in jobInstance['job_types'] %}
		"{{jobType['nameInGo']}}_username": {{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user,
		"{{jobType['nameInGo']}}_password": {{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd,
		{% endfor %}
		// Add other instance creds
		{% for otherJobInstance in vmInstances %}
			{% if otherJobInstance != jobInstance %}
		"{{otherJobInstance['nameInGo']}}_address": {{otherJobInstance['nameInGo']}}Route,
				{% for jobType in otherJobInstance['job_types'] %}
		"{{otherJobInstance.nameInGo}}_{{jobType['nameInGo']}}_username": {{otherJobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user ,
		"{{otherJobInstance.nameInGo}}_{{jobType['nameInGo']}}_password": {{otherJobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd,
				{% endfor %}
			{% endif %}
		{% endfor %}
	}
	{% endfor %}

	// Do deep copy of the service plan properties
	// Modification or addition to job proeprties can be affect the original service plan properties if it was not deep-copied
	{% for jobInstance in vmInstances %}
	additionalProp{{loop.index}}Map := map[string]interface{} {}
	MapDeepCopy(additionalProp{{loop.index}}Map, servicePlan.Properties)
	{{jobInstance['nameInGo']}}InstanceGroup.Properties = additionalProp{{loop.index}}Map
	CopyAdditionalParamsUnderAParentNode({{jobInstance['nameInGo']}}InstanceGroup.Properties, sampleAttributeMap, PARENT_NODE_FOR_ADDN_ATTRS)
	
	{% endfor %}
	
	/*
	if testErrandJob, ok := getJobFromInstanceGroup("test-errand", testErrandInstanceGroup); ok {
		jobTypeInGo2Job.Properties = map[string]interface{}{
			"network": jobTypeInGo2InstanceGroup.Networks[0].Name,
			"address": jobNameInGo2Route,
			"cf": servicePlan.Properties["cf"],
			"jobType1": servicePlan.Properties["jobType1"],
			"jobType2": servicePlan.Properties["jobType2"],
			"username": jobNameInGo2_admin_username,
			"password": jobNameInGo2_admin_password,
			"jobName1_admin_username": jobNameInGo1_admin_username,
			"jobName1_admin_password": jobNameInGo1_admin_password,
		}
	}
	*/

	manifestProperties := map[string]interface{}{
		/*
		"cf": servicePlan.Properties["cf"],
		"jobType1": servicePlan.Properties["jobType1"],
		"jobType2": servicePlan.Properties["jobType2"],
		//"test-errand": servicePlan.Properties["test-errand"],
		*/
	}	

	/* These dont get used anymore in Bosh 2.0 style manifest
	// Global properties are deprecated...
	// Have to repeatedly add them at the job level each time
	for key, val := range servicePlan.Properties {
    	manifestProperties[key] = val
    }
    */

	var updateBlock = bosh.Update{
		Canaries:        {{ vm_updates['canaries'] }},
		MaxInFlight:     {{ vm_updates['max_in_flight'] }},
		CanaryWatchTime: "{{ vm_updates['canary_watch_time'] }}",
		UpdateWatchTime: "{{ vm_updates['update_watch_time'] }}",
		Serial:          boolPointer({{vm_updates['serial'] }}),
	}

	if servicePlan.Update != nil {
		updateBlock = bosh.Update{
			Canaries:        servicePlan.Update.Canaries,
			MaxInFlight:     servicePlan.Update.MaxInFlight,
			CanaryWatchTime: servicePlan.Update.CanaryWatchTime,
			UpdateWatchTime: servicePlan.Update.UpdateWatchTime,
			Serial:          servicePlan.Update.Serial,
		}
	}

	generatedManifest := bosh.BoshManifest{
		Name:     serviceDeployment.DeploymentName,
		Releases: releases,
		Stemcells: []bosh.Stemcell{ {
				Alias:   OnlyStemcellAlias,
				OS:      serviceDeployment.Stemcell.OS,
				Version: serviceDeployment.Stemcell.Version,
			} },
		InstanceGroups: instanceGroups,
		Properties:     manifestProperties,
		Update:         updateBlock,
	}

	manifestBytes, err := yaml.Marshal(generatedManifest)
	if err != nil {
		a.StderrLogger.Printf("[{{product['name']}}] error marshalling bosh manifest: %s", err)
	}

	a.StderrLogger.Printf("[{{product['name']}}] Generated Manifest:\n%s\n----------\n\n", string(manifestBytes))

	return generatedManifest, nil
}

func MapCopy(dst, src interface{}) {
    dv, sv := reflect.ValueOf(dst), reflect.ValueOf(src)

    for _, k := range sv.MapKeys() {
        dv.SetMapIndex(k, sv.MapIndex(k))
    }
}

// Do deep copy of map contents
// Shallow copy leads to changes across all references
func MapDeepCopy(dst, src map[string]interface{}) {
    for key, val := range src {
    	//fmt.Printf("Key: %s, Val is : %v, Type is: %v\n", key, val, reflect.ValueOf(val).Kind())
    	if ( reflect.TypeOf(val).Kind() == reflect.Map) {
    		
    		//fmt.Printf("Calling MapDeepCopy on val : %+v\n", val)
    		newCopy := map[string]interface{}{}
    		MapDeepCopy(newCopy, val.(map[string]interface{}))
    		
    		dst[key] = newCopy
    	} else {
        	dst[key] = val
    	}
    }
}

func CopyAdditionalParamsUnderAParentNode(destnAttributeMap, srcAttributeMap map[string]interface{}, parentNode string) {
	existingNodeMap := destnAttributeMap[parentNode]

	if (existingNodeMap == nil) {
		srcAttributeMap[parentNode] = srcAttributeMap
	} else {
		existingStringMap, _ := existingNodeMap.(map[string]interface{})
		for key, val := range srcAttributeMap {
			// Assuming the value type is string
			valStr := val.(string)
			// Append the new value coming from the request to exisitng set
			existingVal := existingStringMap[key]
			if ((existingVal != nil) && (existingVal.(string) != "")) {
				valStr = existingVal.(string) + "," + val.(string)					
			}
  			existingStringMap[key] =  valStr				
  		}
	}
}

// Override
func UpdateManifest(serviceDeployment serviceadapter.ServiceDeployment,
	servicePlan serviceadapter.Plan,
	requestParams serviceadapter.RequestParameters,
	previousManifest *bosh.BoshManifest,
	previousPlan *serviceadapter.Plan,
) (bosh.BoshManifest, error) {

	// NOP
	// Change code if update has to change the manifest using request params or changed plan type etc..
	return *previousManifest, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getJobFromInstanceGroup(name string, instanceGroup *bosh.InstanceGroup) (*bosh.Job, bool) {
	for index, job := range instanceGroup.Jobs {
		if job.Name == name {
			return &instanceGroup.Jobs[index], true
		}
	}
	return &bosh.Job{}, false
}

func instanceCounts(plan serviceadapter.Plan) map[string]int {
	val := map[string]int{}
	for _, instanceGroup := range plan.InstanceGroups {
		val[instanceGroup.Name] = instanceGroup.Instances
	}
	return val
}

func boolPointer(b bool) *bool {
	return &b
}

func checkInstanceGroupsPresent(names []string, instanceGroups []serviceadapter.InstanceGroup) error {
	var missingNames []string

	for _, name := range names {
		if !containsInstanceGroup(name, instanceGroups) {
			missingNames = append(missingNames, name)
		}
	}

	if len(missingNames) > 0 {
		return fmt.Errorf("Invalid instance group configuration: expected to find: '%s' in list: '%s'",
			strings.Join(missingNames, ", "),
			strings.Join(getInstanceGroupNames(instanceGroups), ", "))
	}
	return nil
}

func getInstanceGroupNames(instanceGroups []serviceadapter.InstanceGroup) []string {
	var instanceGroupNames []string
	for _, instanceGroup := range instanceGroups {
		instanceGroupNames = append(instanceGroupNames, instanceGroup.Name)
	}
	return instanceGroupNames
}

func containsInstanceGroup(name string, instanceGroups []serviceadapter.InstanceGroup) bool {
	for _, instanceGroup := range instanceGroups {
		if instanceGroup.Name == name {
			return true
		}
	}

	return false
}
