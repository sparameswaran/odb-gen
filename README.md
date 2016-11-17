# service-adapter-gen

Generate ODB Service Adapter release using pre-built templates
Read more about on-demand service broker [here](https://docs.pivotal.io/on-demand-service-broker/)
The ODB allows creation of dedicated managed service instances managed by Bosh on the fly. It depends on the ODB bosh release, an implementation of a service adapter and actual managed service bosh release. The realtime bosh manifest and binding creation are customized via the service adapter to match the requirements of the managed service.

The service adapter code is responsible for the actual manifest generation based on the type of service plan selected by the user. It can fine tune the number of instances, vm instances and job properties as part of the manifest generation. The create binding allows the appropriate flags, parameters to be passed back to the user who requested the service instance.

The tools in this repo help generate the basic saffolding for a customized service adapter and also related artifacts for tile and bosh release generation.

# Requirements:
* Latest On Demand Broker Bosh release (download from https://network.pivotal.io)
* Bosh release of the managed service
* golang setup

# Generating the service adapter

## Create and customize adapter.yml
* Clone the service-adapter-gen repo in local env
* Create an adapter.yml file in the root of the repo. Use the `templates/sample-adapter.yml` as an example adapter.yml
* Customize/edit the name of the product, tile, releases
* Specify the various dependent releases (like main managed service, a separate service-backup, consul release etc.) that constitute the overall managed service. Specify the jobs that form part of the managed release.
* Specify the vms that would be running the different jobs within them. The job types specified should match the job names specified against the releases.
* Edit the Service and service plans to change number of instances of various VMs.
* Create a resources folder and Copy the bosh releases (for ODB, Managed service) along with the image icon file into resources folder. To do a quick run, just simply use touch to create empty files with matching names under resources folder.
* Ensure the names and path match with the various resources in the adapter.yml.

## Build
* Run ${REPO_ROOT}/bin/adapter-gen build
* This should generate a customized service adapter code and set of templates/scripts to help test the adapter, all under release folder.
* The service adapter golang code would be saved under release/<product>-service-adapter folder.
* The adapter test scripts will be saved under release/test-adapter-stubs folder.
* The release and tile creation scripts will be saved under release/scripts
* The tile metadata related files would be saved under tile-templates
* Sample bosh-lite warden manifest would be saved under manifests

## Testing the generated adapter code
* Setup GOPATH env variable to point to existing GO repo.
* Change to the release folder
* Source the go.env ( `. ./go.env`)
* Change to the test-adapter-stubs directory 
* Run ./genManfest.sh
  genManifest.sh tries to invoke the service adapter's generate_manifest.go and generate the bosh manifest that conforms to the service plan guidelines and the vms described in the adapter.yml input file. The output gets saved as manifest.yml in the same test-adapter-stubs folder.
* Run ./convertYml2Json.sh 
  This would generate a json representation of the manifest based on input yaml file.A
  Run the command with input and output file names.
  Example: ./convertYml2Json.sh manifest.yml manifest.json
* Run ./createBinding.sh
  createBinding.sh tries to invoke the service adapter's create_binding.go and generate the credentials that would be part of the service bind call. 
* To test update of a service based on newer request parameters (either instances or different set of parameters), run ./updateExistingManifest.sh
  This would attempt to reuse the previously generated manifest.json (converted from manifest.yml by running convertYml2Json.sh script just before createBindind) and use additional modified request parameters to see how the manifest update occurs. The code does not take into account newer plan specification for an existing service (but can be achieved if needed).

## Understanding and customizing the service adapter code
* generate_manifest.go code attempts to iterate over each vm instance and add properties like username/password for every job running within that vm  The username and passwords are randomly generated. The code also tries to expose those generated values to other vm via properties set. Feel free to edit/add/remove/change any additional properties that need to be set within the jobs. Then re-run the genManifest.sh script to generate the changed manifest.
* create_binding.go code pulls the various properties from the bosh manifest (either from vm level or other known properteis) and present it as a map to the user. The code also tries to provide the various vm instance addresses. Feel free to edit/add/remove/change any additional properties that need to exposed to the user. Then re-run the createBinding.sh script to test the changed behavior.

## Release and tile creation
* The tile metadata gets saved under tile-templates folder of the generated release folder 
* Based on the service plans defined in the adapter.yml, forms for different plans would be automatically created in the tile metadata file while expposing the vm type, persistent disk spaces, instances and AZs to be controlled by the user.
* Edit the auto-generated metadata file to add/remove any necessary parameters/values, types as necessary to match the requirements.
* Use the scripts/createTile.sh or scripts/createRelease.sh to generate the tile metadata and bosh release after customizing the adapter or tile metadata.
