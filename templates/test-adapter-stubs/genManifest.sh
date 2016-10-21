#!/bin/bash
#set -xv
SCRIPT_DIR=$(dirname $0)
if [ "$(pwd)" != "$SCRIPT_DIR" -a "." != "$SCRIPT_DIR" ]; then
	echo "genManifest.sh should be run from test-adapter-stubs folder"
	exit -1
fi

. $SCRIPT_DIR/../go.env


./gen_manifest.py generate-manifest | tee manifest.yml

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
  echo "Generated manifest output saved as manifest.yml"
  echo "Run ./convertYml2Json.sh before running ./createBinding.sh"
fi

echo ""
