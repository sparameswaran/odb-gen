#!/usr/bin/env python

# tile-generator
#
# Copyright (c) 2015-Present Pivotal Software, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import sys
import errno
import glob
import requests
import shutil
import stat
import subprocess
import tarfile
import template
import urllib
import zipfile
import yaml
import re
import datetime

from bosh import *
from util import *
from subprocess import Popen, PIPE, STDOUT


LIB_PATH = os.path.dirname(os.path.realpath(__file__))
REPO_PATH = os.path.realpath(os.path.join(LIB_PATH, '..'))
DOCKER_BOSHRELEASE_VERSION = '23'

def dumpclean(obj):
    if type(obj) == dict:
        for k, v in obj.items():
            if hasattr(v, '__iter__'):
                print k
                dumpclean(v)
            else:
                print '%s : %s' % (k, v)
    elif type(obj) == list:
        for v in obj:
            if hasattr(v, '__iter__'):
                dumpclean(v)
            else:
                print v
    else:
        print obj

def copyanything(src, dst):
    try:
        shutil.copytree(src, dst)
    except OSError as exc: # python >2.5
        if exc.errno == errno.ENOTDIR:
            shutil.copy(src, dst)
        else: raise

def build(config, verbose=False):
    # Clean up config (as above)
    context = config.copy()
    #add_defaults(context)
    context['verbose'] = verbose
    validate_config(context)
    
    with cd('release', clobber=True):
        gen_release_dir('config', context)
        gen_jobs('jobs', context)
        gen_adapter('src', context)
        gen_manifest('manifests', context)
        gen_packages('packages', context)
        gen_tile_metadata('tile-templates', context)
        gen_scripts('scripts', context)
        gen_test_stubs('test-adapter-stubs', context)  
        #add_golang(context)
        #bosh('upload', 'blobs')
        output = bosh('create', 'release', '--force', '--final', '--with-tarball', '--version', context['version'])
        context['release'] = bosh_extract(output, [
            { 'label': 'name', 'pattern': 'Release name' },
            { 'label': 'version', 'pattern': 'Release version' },
            { 'label': 'manifest', 'pattern': 'Release manifest' },
            { 'label': 'tarball', 'pattern': 'Release tarball' },
        ])
        gen_tile('tile-output', context)


def validate_config(context):
    context['root_dir'] = os.getcwd()
    name = context['product']['name'].lower().replace(' ', '-')   
    if (name.endswith('service-adapter') != True):
        name += '-service-adapter'

    context['product']['name'] = name
    context['product']['short_name'] = context['product']['short_name'].lower().replace(' ', '-')
    context['product']['title'] = context['product']['short_name'].title()
    
    context['odb_release']['release_file'] = os.path.basename(context['odb_release']['file'])

    validate_managed_services_config(context)
    validate_vms_config(context)
    validate_plan_config(context)
    validate_vm_updates_config(context)

def validate_managed_services_config(context):
    managed_services_jobs = []
    job2release_lookup_map = { }
    for managed_service_release in context['managed_service_releases']:
        managed_service_release['release_file'] = os.path.basename(managed_service_release['file'])
        managed_service_jobs = managed_service_release['jobs']
        for job_name in managed_service_jobs:
            job2release_lookup_map[job_name] = managed_service_release['name']
        managed_services_jobs += managed_service_jobs
        jobs_as_str = str(managed_service_jobs)
        managed_service_release['jobs_as_str'] = jobs_as_str.replace('\'', '"')
    
    context['managed_services_jobs'] =  managed_services_jobs
    context['job2release_lookup_map'] = job2release_lookup_map

def validate_vms_config(context):
    index = 0
    for vm in context['vms']:
        #dumpclean(vm)
        vm['name_lower'] = vm['name'].lower()
        vm['nameInGo']   = vm['name_lower'].replace('-','_')

        jobIndex = 0
        vm['job_types_in_go'] = [ ]
        vm['job_types_as_str'] = ''
        for jobType in vm['job_types']:
            typeInGo = jobType['name'].lower().replace('-','_')
            jobType['nameInGo'] = typeInGo
            vm['job_types_in_go'].append(typeInGo)
            vm['job_types_as_str'] += '"' + jobType['name'] + '", '
            jobType['release'] = context['job2release_lookup_map'][jobType['name']]
            
        
        vm['job_types_as_str'] = vm['job_types_as_str'][:-2]
        #print 'Job Type as string array: ' + vm['job_types_as_str'] + "$"
        vm['index'] = index
        index += 1

def validate_plan_config(context):
    for plan in context['service']['service_plans']:
        plan['title'] = plan['name'].title()
        plan['name'] = plan['name'].lower().replace(' ', '-')
        plan['nameInGo'] = plan['name'].replace('-', '_')
        for vm in plan['vms']:
            vm['name_lower'] = vm['name'].lower()
            vm['nameInGo']   = vm['name_lower'].replace('-','_')
            vm['properties'] = {}
            vm['properties']['type'] = plan['name']
            #dumpclean(vm['properties'])

def validate_vm_updates_config(context): 
    vm_updates = context.get('vm_updates', {})
    vm_updates['canaries']          = vm_updates.get('canaries', 1)
    vm_updates['max_in_flight']     = vm_updates.get('max_in_flight', 10)
    vm_updates['canary_watch_time'] = vm_updates.get('canary_watch_time', '30000-240000')
    vm_updates['update_watch_time'] = vm_updates.get('update_watch_time', '30000-240000')
    vm_updates['serial']            = str(vm_updates.get('serial', 'true')).lower()

    context['vm_updates'] = vm_updates
   
def gen_adapter(dir, context, alternate_template=None):
    adapter_dir = os.path.realpath(os.path.join(dir, context['product']['name'] ))
    #print 'Going to copy from ' + context['root_dir'] + '/templates/src/service-adapter into adapter-dir: ' + adapter_dir
    
    shutil.copytree(context['root_dir'] + '/templates/src/service-adapter/',  adapter_dir)   

    template_dir = 'src/service-adapter'
    

    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)
    adapter_context = {
        'context': context,
        'vmInstances': context['vms'],
        'product': context['product'],
        'vm_updates': context['vm_updates'],
        'job2release_lookup_map': context['job2release_lookup_map'],
        'files': []
    }

    template.render(
        os.path.join(adapter_dir, 'adapter/generate_manifest.go'),
        os.path.join(template_dir, 'adapter/generate_manifest.go'),
        adapter_context
    )
    template.render(
        os.path.join(adapter_dir, 'adapter/create_binding.go'),
        os.path.join(template_dir, 'adapter/create_binding.go'),
        adapter_context
    )
    template.render(
        os.path.join(adapter_dir, 'adapter/generate_dashboard_url.go'),
        os.path.join(template_dir, 'adapter/generate_dashboard_url.go'),
        adapter_context
    )
    
    template.render(
        os.path.join(adapter_dir, 'adapter/adapter.go'),
        os.path.join(template_dir, 'adapter/adapter.go'), 
        adapter_context
    )

    template.render(
        os.path.join(adapter_dir, 'adapter/delete_binding.go'), 
        os.path.join(template_dir, 'adapter/delete_binding.go'), 
        adapter_context
    )

    template.render(
        os.path.join(adapter_dir, 'cmd/service-adapter/main.go'),
        os.path.join(template_dir, 'cmd/service-adapter/main.go'),
        adapter_context
    )

    print 'Done rendering templates at:' + adapter_dir

def gen_manifest(dir, context, alternate_template=None):
    manifest_dir = os.path.realpath(os.path.join(dir ))
    
    mkdir_p(manifest_dir)
    template_dir = 'manifests'
    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)
    manifest_context = {
        'context': context,
        'vmInstances': context['vms'],
        'product': context['product'],
        'service': context['service'],
        'job2release_lookup_map': context['job2release_lookup_map'],
        'managed_service_releases': context['managed_service_releases'],
        #'managed_service_release_jobs': context['managed_service_release_jobs'],
        'files': []
    }
    
    shutil.copy(context['root_dir'] + '/templates/manifests/bosh-lite-cloud-config.yml',  manifest_dir)
    

    template.render(
        os.path.join(manifest_dir, context['product']['short_name'] + '-odb-warden-manifest.yml'),
        os.path.join(template_dir, 'on-demand-broker-warden-manifest.yml' ),
        manifest_context
    )

def gen_release_dir(dir, context, alternate_template=None):
    release_dir = os.path.realpath(os.path.join(dir ))
    
    template_dir = dir
    release_context = {
        'context': context,
        'product': context['product'],
        #'managed_service_release_jobs': context['managed_service_release_jobs'],
        'files': []
    }
    template.render(
        os.path.join(release_dir, 'blobs.yml'),
        os.path.join(template_dir, 'blobs.yml' ),
        release_context
    )
    template.render(
        os.path.join(release_dir, 'final.yml'),
        os.path.join(template_dir, 'final.yml' ),
        release_context
    )    

def gen_jobs(dir, context, alternate_template=None):
    jobs_dir = os.path.realpath(os.path.join(dir ))
    
    template_dir = dir
    jobs_context = {
        'context': context,
        'product': context['product'],
        #'managed_service_release_jobs': context['managed_service_release_jobs'],
        'files': []
    }
    template.render(
        os.path.join(jobs_dir, context['product']['name'] + '/monit'),
        os.path.join(template_dir, 'service-adapter/monit' ),
        jobs_context
    )
    template.render(
        os.path.join(jobs_dir, context['product']['name'] + '/spec'),
        os.path.join(template_dir, 'service-adapter/spec' ),
        jobs_context
    )    

def gen_packages(dir, context, alternate_template=None):
    packages_dir = os.path.realpath(os.path.join(dir ))
    
    mkdir_p(packages_dir)
    template_dir = dir
    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)
    packages_context = {
        'context': context,
        'product': context['product'],
        #'managed_service_release_jobs': context['managed_service_release_jobs'],
        'files': []
    }    

    template.render(
        os.path.join(packages_dir, 'go/packaging'),
        os.path.join(template_dir, 'go/packaging' ),
        packages_context
    )
    template.render(
        os.path.join(packages_dir, 'go/spec'),
        os.path.join(template_dir, 'go/spec' ),
        packages_context
    )
    template.render(
        os.path.join(packages_dir, 'odb-service-adapter/spec'),
        os.path.join(template_dir, 'odb-service-adapter/spec' ),
        packages_context
    )
    template.render(
        os.path.join(packages_dir, 'odb-service-adapter/packaging'),
        os.path.join(template_dir, 'odb-service-adapter/packaging' ),
        packages_context
    )

def add_golang(context, alternate_template=None):
    add_blob_package(context,
        {
            'name': 'go',
            'files': [{
                'name': 'go1.7.1.linux-amd64.tar.gz',
                'path': 'https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz'
            }]
        }, alternate_template='go'
    )


def gen_tile_metadata(dir, context, alternate_template=None):
    tile_dir = os.path.realpath(os.path.join(dir ))
    context['product']['tile']['version'] = context['version']
    
    mkdir_p(tile_dir)
    template_dir = 'tile'
    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)

    service = context['service']
    for service_plan in service['service_plans']:
        service_plan['title'] = service_plan['name'].title()
    
    dumpclean(context['product']['tile'])

    tile_context = {
        'context': context,
        'product': context['product'],
        'history': context['history'],
        'version': context['version'],
        'tile'   : context['product']['tile'],
        'service': context['service'],
        'odb_release':  context['odb_release'],
        'job2release_lookup_map': context['job2release_lookup_map'],
        'managed_service_releases': context['managed_service_releases'],
        'files': []
    }

    template.render(
        os.path.join(tile_dir, 'content-migrations.yml'),
        os.path.join(template_dir, 'content-migrations.yml' ),
        tile_context
    )
    template.render(
        os.path.join(tile_dir, 'migration.js'),
        os.path.join(template_dir, 'migration.js' ),
        tile_context
    )
    template.render(
        os.path.join(tile_dir, context['product']['short_name'] + '-odb-service-tile.yml'),
        os.path.join(template_dir, 'odb-service-tile.yml'),
        tile_context
    )

def gen_test_stubs(dir, context, alternate_template=None):
    test_stubs_dir = os.path.realpath(os.path.join(dir ))
    context['product']['tile']['version'] = context['version']
    
    mkdir_p(test_stubs_dir)
    template_dir = dir
    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)
        
    stubs_context = {
        'context' : context,
        'root_dir': context['root_dir'],
        'product' : context['product'],
        'history' : context['history'],
        'version' : context['version'],
        'tile'    : context['product']['tile'],
        'service' : context['service'],
        'service' : context['service'],
        'odb_release':  context['odb_release'],
        'job2release_lookup_map': context['job2release_lookup_map'],
        'managed_service_releases': context['managed_service_releases'],
        'files': []
    }
    
    template.render(
        os.path.join(test_stubs_dir, 'bosh-vms.json'),
        os.path.join(template_dir, 'bosh-vms.json' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'create_test_binding.py'),
        os.path.join(template_dir, 'create_test_binding.py' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'gen_manifest.py'),
        os.path.join(template_dir, 'gen_manifest.py' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'update_manifest.py'),
        os.path.join(template_dir, 'update_manifest.py' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'deployment.json'),
        os.path.join(template_dir, 'deployment.json' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'plan.json'),
        os.path.join(template_dir, 'plan.json' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'request.json'),
        os.path.join(template_dir, 'request.json' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'request2.json'),
        os.path.join(template_dir, 'request2.json' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'sample_manifest.json'),
        os.path.join(template_dir, 'sample_manifest.json' ),
        stubs_context
    )
    template.render(
        os.path.join(test_stubs_dir, 'sample_manifest.yml'),
        os.path.join(template_dir, 'sample_manifest.yml' ),
        stubs_context
    )
    
    shutil.copy(context['root_dir'] + '/templates/go.env',  '.')
    shutil.copy(context['root_dir'] + '/templates/' + template_dir + '/createBinding.sh',  test_stubs_dir)
    shutil.copy(context['root_dir'] + '/templates/' + template_dir + '/convertYml2Json.sh',  test_stubs_dir)
    shutil.copy(context['root_dir'] + '/templates/' + template_dir + '/genManifest.sh',  test_stubs_dir)
    shutil.copy(context['root_dir'] + '/templates/' + template_dir + '/updateExistingManifest.sh',  test_stubs_dir)
    
    for scriptFile in glob.glob(test_stubs_dir + '/*.??'):
        fileStat = os.stat(scriptFile)
        os.chmod(scriptFile,  fileStat.st_mode | stat.S_IEXEC)

def gen_scripts(dir, context, alternate_template=None):
    scripts_dir = os.path.realpath(os.path.join(dir ))
    context['product']['tile']['version'] = context['version']
    
    mkdir_p(scripts_dir)
    template_dir = dir
    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)

    service = context['service']
    for service_plan in service['service_plans']:
        service_plan['title'] = service_plan['name'].title()
        
    tile_context = {
        'context': context,
        'root_dir': context['root_dir'],
        'product': context['product'],
        'history': context['history'],
        'version': context['version'],
        'tile'   : context['product']['tile'],
        'service': context['service'],
        'odb_release':  context['odb_release'],
        'managed_service_releases': context['managed_service_releases'],
        'files': []
    }
    

    template.render(
        os.path.join(scripts_dir, 'createRelease.sh'),
        os.path.join(template_dir, 'createRelease.sh' ),
        tile_context
    )
    template.render(
        os.path.join(scripts_dir, 'createTile.sh'),
        os.path.join(template_dir, 'createTile.sh' ),
        tile_context
    )

    tileScript = os.stat(scripts_dir + '/createTile.sh')
    os.chmod(scripts_dir + '/createTile.sh', tileScript.st_mode | stat.S_IEXEC)
    releaseScript = os.stat(scripts_dir + '/createRelease.sh')
    os.chmod(scripts_dir + '/createRelease.sh', releaseScript.st_mode | stat.S_IEXEC)


def gen_tile(dir, context, alternate_template=None):
    release_dir = os.getcwd()
    tile_output_dir = os.path.realpath(os.path.join(dir ))
    #print './scripts/createTile.sh', ' '.join([release_dir, tile_output_dir])
    command = [ './scripts/createTile.sh'] + [ release_dir, tile_output_dir ]
    try:
        #return subprocess.check_output(command, stderr=subprocess.STDOUT, cwd=release_dir)
        cmd = './scripts/createTile.sh ' + release_dir + ' ' + tile_output_dir 
        p = Popen(cmd, shell=True, stdin=PIPE, stdout=PIPE, stderr=STDOUT, close_fds=True)
        output = p.stdout.read()
        print output
    except subprocess.CalledProcessError as e:
        print e.output
        sys.exit(e.returncode)


def add_blob_package(context, package, alternate_template=None):
    add_package('blobs', context, package, alternate_template)

# FIXME dead code, remove.
def add_package(dir, context, package, alternate_template=None):
    name = package['name'].lower().replace('-','_')
    package['name'] = name
    bosh('generate', 'package', name)
    target_dir = os.path.realpath(os.path.join(dir, name))
    package_dir = os.path.realpath(os.path.join('packages', name))
    mkdir_p(target_dir)
    template_dir = 'packages'
    if alternate_template is not None:
        template_dir = os.path.join(template_dir, alternate_template)
    package_context = {
        'context': context,
        'package': package,
        'files': []
    }
    with cd('..'):
        files = package.get('files', [])
        path = package.get('path', None)
        if path is not None:
            files += [ { 'path': path } ]
            package['path'] = os.path.basename(path)
        manifest = package.get('manifest', None)
        manifest_path = None
        if type(manifest) is dict:
            manifest_path = manifest.get('path', None)
        if manifest_path is not None:
            files += [ { 'path': manifest_path } ]
            package['manifest']['path'] = os.path.basename(manifest_path)
        for file in files:
            filename = file.get('name', os.path.basename(file['path']))
            file['name'] = filename
            urllib.urlretrieve(file['path'], os.path.join(target_dir, filename))
            package_context['files'] += [ filename ]
        for docker_image in package.get('docker_images', []):
            filename = docker_image.lower().replace('/','-').replace(':','-') + '.tgz'
            download_docker_image(docker_image, os.path.join(target_dir, filename), cache=context.get('docker_cache', None))
            package_context['files'] += [ filename ]
    if package.get('is_app', False):
        manifest = package.get('manifest', { 'name': name })
        if manifest.get('random-route', False):
            print >> sys.stderr, 'Illegal manifest option in package', name + ': random-route is not supported'
            sys.exit(1)
        manifest_file = os.path.join(target_dir, 'manifest.yml')
        with open(manifest_file, 'wb') as f:
            f.write('---\n')
            f.write(yaml.safe_dump(manifest, default_flow_style=False))
        package_context['files'] += [ 'manifest.yml' ]
        update_memory(context, manifest)
    template.render(
        os.path.join(package_dir, 'spec'),
        os.path.join(template_dir, 'spec'),
        package_context
    )
    template.render(
        os.path.join(package_dir, 'packaging'),
        os.path.join(template_dir, 'packaging'),
        package_context
    )


# FIXME dead code, remove.
def bosh(*argv):
    argv = list(argv)
    print 'bosh', ' '.join(argv)
    command = [ 'bosh', '--no-color', '--non-interactive' ] + argv
    try:
        return subprocess.check_output(command, stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError as e:
        if argv[0] == 'init' and argv[1] == 'release' and 'Release already initialized' in e.output:
            return e.output
        if argv[0] == 'generate' and 'already exists' in e.output:
            return e.output
        print e.output
        sys.exit(e.returncode)

# FIXME dead code, remove.
def bash(*argv):
    argv = list(argv)
    try:
        return subprocess.check_output(argv, stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError as e:
        print ' '.join(argv), 'failed'
        print e.output
        sys.exit(e.returncode)

def is_semver(version):
    valid = re.compile('[0-9]+\\.[0-9]+\\.[0-9]+([\\-+][0-9a-zA-Z]+(\\.[0-9a-zA-Z]+)*)*$')
    return valid.match(version) is not None

def is_unannotated_semver(version):
    valid = re.compile('[0-9]+\\.[0-9]+\\.[0-9]+$')
    return valid.match(version) is not None

def update_version(history, version):
    if version is None:
        version = 'patch'
    prior_version = history.get('version', None)
    if prior_version is not None:
        history['history'] = history.get('history', [])
        history['history'] += [ prior_version ]
    if not is_semver(version):
        semver = history.get('version', '0.0.0')
        if not is_unannotated_semver(semver):
            print >>sys.stderr, 'The prior version was', semver
            print >>sys.stderr, 'To auto-increment, the prior version must be in semver format (x.y.z), and must not include a label.'
            sys.exit(1)
        semver = semver.split('.')
        if version == 'patch':
            semver[2] = str(int(semver[2]) + 1)
        elif version == 'minor':
            semver[1] = str(int(semver[1]) + 1)
            semver[2] = '0'
        elif version == 'major':
            semver[0] = str(int(semver[0]) + 1)
            semver[1] = '0'
            semver[2] = '0'
        else:
            print >>sys.stderr, 'Argument must specify "patch", "minor", "major", or a valid semver version (x.y.z)'
            sys.exit(1)
        version = '.'.join(semver)
    history['version'] = version
    return version
