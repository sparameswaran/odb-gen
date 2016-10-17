#!/usr/bin/env python

from subprocess import call
from sys import argv
from os import system

with open("./deployment.json", "r") as myfile:
    bosh_info = myfile.read().replace('\n', ' ').replace('\r', '')

#with open("./params.json", "r") as myfile:
#    params = myfile.read().replace('\n', ' ').replace('\r', '')

with open("./plan.json", "r") as myfile:
    plan = myfile.read().replace('\n', ' ').replace('\r', '')

with open("./request.json", "r") as myfile:
    requestParams = myfile.read().replace('\n', ' ').replace('\r', '')

#with open("./service-releases.json", "r") as myfile:
#    service_releases = myfile.read().replace('\n', ' ').replace('\r', '')
call(['go', 'run', '../src/{{product['short_name']}}-service-adapter/cmd/service-adapter/main.go', argv[1], bosh_info, plan, requestParams, '---', '{}'])

