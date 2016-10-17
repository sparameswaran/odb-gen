echo ""
echo "Before running make sure the manifest.json contains updated json content"
echo "First capture the manifest in yml format from testManifest and then convert yaml to json"
echo "Use convertYml2Json.sh or link to convert: http://codebeautify.org/yaml-to-json-xml-csv" 
echo ""
. ../go.env 

if [ $# -eq 1]
	input=$1
else
  	input="manifest.json"
fi

if [ -e "$input" ]; then
	echo "Using the converted manifest.json for testing create_test_binding"
else
	echo "Didnt find any manifest.json or specified input for testing create_test_binding"
	input="sample_manifest.json"
	echo "Using the sample_manifest.json for testing create_test_binding"
	echo "Save output of ./genManifest.sh and use convertYml2Json.sh to generate the manifest.json"
fi

./create_test_binding.py create-binding $input
