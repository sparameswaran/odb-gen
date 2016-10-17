. ../go.env  >/dev/null
./gen_manifest.py generate-manifest | tee manifest.yml

if [ $? -ne 0 ]; then
	echo "Error with manifest generation"
	echo "Install posixpath if its missing!!"
	echo "Use: python -m pip install posixpath"
else
  echo ""
  echo "Generated manifest output saved as manifest.yml"
  echo "Run convertYml2Json.sh before running createBinding.sh"
fi

echo ""
