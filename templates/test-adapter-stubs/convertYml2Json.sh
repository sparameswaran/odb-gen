#!/bin/bash

if [ $# -ne 2 ]; then
  echo "Provide 2 arguments: <input-yaml-file> <output-yaml-filename>"
  exit -1
fi

input=$1
output=$2
echo ""
echo "Converting input yml file: $input to output json file: $output"
python -c 'import sys, yaml, json; json.dump(yaml.load(sys.stdin), sys.stdout, indent=2)' < $input > $output
echo "" >> $output
echo ""
