package adapter

import (
	"bytes"
	"os/exec"
	"fmt"
	"reflect"
	"strings"

	"github.com/pivotal-cf/on-demand-services-sdk/bosh"
	"github.com/pivotal-cf/on-demand-services-sdk/serviceadapter"
)

func (b *Binder) CreateBinding(bindingId string, boshVMs bosh.BoshVMs, manifest bosh.BoshManifest, requestParams serviceadapter.RequestParameters) (serviceadapter.Binding, error) {
	//params := requestParams.ArbitraryParams()

	/*
	var invalidParams []string
	for paramKey, _ := range params {
		if paramKey != "topic" {
			invalidParams = append(invalidParams, paramKey)
		}
	}

	if len(invalidParams) > 0 {
		sort.Strings(invalidParams)
		errorMessage := fmt.Sprintf("unsupported parameter(s) for this service: %s", strings.Join(invalidParams, ", "))
		b.StderrLogger.Println(errorMessage)
		return serviceadapter.Binding{}, errors.New(errorMessage)
	}
	*/

	/*
	jobNameInGo1Hosts := boshVMs["jobName1"]
	if len(jobNameInGo1Hosts) == 0 {
		b.StderrLogger.Println("no VMs for instance group jobName1")
		return serviceadapter.Binding{}, errors.New("")
	}

	jobNameInGo1HostRoute = ""

	var kafkaAddresses []interface{}
	for _, kafkaHost := range kafkaHosts {
		kafkaAddresses = append(kafkaAddresses, fmt.Sprintf("%s:9092", kafkaHost))
	}

	zookeeperServers := boshVMs["zookeeper_server"]
	if len(zookeeperServers) == 0 {
		b.StderrLogger.Println("no VMs for job zookeeper_server")
		return serviceadapter.Binding{}, errors.New("")
	}

	if _, errorStream, err := b.Run(b.TopicCreatorCommand, strings.Join(zookeeperServers, ","), bindingId); err != nil {
		if strings.Contains(string(errorStream), "kafka.common.TopicExistsException") {
			b.StderrLogger.Println(fmt.Sprintf("topic '%s' already exists", bindingId))
			return serviceadapter.Binding{}, serviceadapter.NewBindingAlreadyExistsError()
		}
		b.StderrLogger.Println("Error creating topic: " + err.Error())
		return serviceadapter.Binding{}, errors.New("")
	}

	if params["topic"] != nil {
		if _, _, err := b.Run(b.TopicCreatorCommand, strings.Join(zookeeperServers, ","), params["topic"].(string)); err != nil {
			b.StderrLogger.Println("Error creating topic: " + err.Error())
			return serviceadapter.Binding{}, errors.New("")
		}
	}
	*/


	instanceGroups := manifest.InstanceGroups
	
    cfDomainRoute := ""
    servicePlanType := ""

	if (servicePlanType == "") {
		servicePlanType = instanceGroups[0].Properties["type"].(string)
	}

	cfMap := instanceGroups[0].Properties["cf"]
	
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

    {% for jobInstance in vmInstances %}
	{{jobInstance['nameInGo']}}_vm :=  instanceGroups[{{loop.index0}}]
    {{jobInstance['nameInGo']}}vm_route := {{jobInstance['nameInGo']}}_vm.Properties["address"].(string)
    {{jobInstance['nameInGo']}}vm_hosts := strings.Join(boshVMs["{{jobInstance['name']}}"], ",")
	{% for jobType in jobInstance['job_types'] %}
	{{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user := {{jobInstance['nameInGo']}}_vm.Properties["{{jobType['nameInGo']}}_username"].(string)
	{{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd := {{jobInstance['nameInGo']}}_vm.Properties["{{jobType['nameInGo']}}_password"].(string)
	{% endfor %}

	{% endfor %}

	arbitraryParameters := requestParams.ArbitraryParams()

	b.StderrLogger.Printf("[{{product['name']}}] CreateBinding with arbitraryParameters: \n%+v\n", arbitraryParameters)

	generatedBinding := serviceadapter.Binding{
		Credentials: map[string]interface{}{
			"service_type": servicePlanType,
			{% for jobInstance in vmInstances %}
				"{{jobInstance.nameInGo}}_url": fmt.Sprintf("https://%s.%s", {{jobInstance['nameInGo']}}vm_route, cfDomainRoute),
				"{{jobInstance['nameInGo']}}_hosts": {{jobInstance['nameInGo']}}vm_hosts,				
				{% for jobType in jobInstance['job_types'] %}
				"{{jobInstance.nameInGo}}_{{jobType['nameInGo']}}_username": {{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_user,
				"{{jobInstance.nameInGo}}_{{jobType['nameInGo']}}_password": {{jobInstance['nameInGo']}}vm_{{jobType['nameInGo']}}_passwd,
				{% endfor %}
			{% endfor %}
		},
	}

	b.StderrLogger.Printf("\n[{{product['name']}}] DEBUG Generated Binding Creds:\n %+v\n\n", generatedBinding)
	return generatedBinding, nil
}

//go:generate counterfeiter -o fake_command_runner/fake_command_runner.go . CommandRunner
type CommandRunner interface {
	Run(name string, arg ...string) ([]byte, []byte, error)
}

type ExternalCommandRunner struct{}

func (c ExternalCommandRunner) Run(name string, arg ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, arg...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.Output()
	return stdout, stderr.Bytes(), err
}
