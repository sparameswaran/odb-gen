#!/bin/bash
#set -xv
SCRIPT_DIR=$(dirname $0)
if [ "$(pwd)" != "$SCRIPT_DIR" -a "." != "$SCRIPT_DIR" ]; then
	echo "updateExistingManifest.sh should be run from test-adapter-stubs folder and also requires a manifest.json file"
	exit -1
fi

. $SCRIPT_DIR/../go.env


./update_manifest.py generate-manifest | tee manifest2.yml

if [ $? -ne 0 ]; then
	echo "Error with manifest generation"
	echo ""
	echo "Install posixpath if its missing!!"
	echo "Use: python -m pip install posixpath"
	echo ""
	echo "If problem in locating the vendor go source files,"
	echo " go to the release folder (one level up), ensure $GOPATH is set and run '. ./go.env'"
	echo ""
else
  echo ""
  echo "Generated manifest output saved as manifest2.yml"
  echo "Run ./convertYml2Json.sh before running ./createBinding.sh"
fi

echo ""
