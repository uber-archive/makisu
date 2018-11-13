import subprocess
import pytest
import os
import requests
import shutil
import tempfile
import time
import utils

REGISTRY_IMAGE = "registry:2.4.1"
REGISTRY1_PORT = 5002
REGISTRY2_PORT = 5003
REGISTRY1_CONTAINER = "test-registry1"
REGISTRY2_CONTAINER = "test-registry2"
REGISTRY1_CONFIG_PATH = os.path.join(os.getcwd(), "integration/python/config/registry1.yaml")
REGISTRY2_CONFIG_PATH = os.path.join(os.getcwd(), "integration/python/config/registry2.yaml")


class Registry(object):

    def __init__(self, port, name, config):
        self.name = name
        self.port = port
        self.stop()
        utils.docker_pull_image(REGISTRY_IMAGE)
        print 'Starting container ' + self.name
        run_cmd = [
            'docker', 'run',
            '-d',
            '--name=' + self.name,
            '-p', '{o}:{i}'.format(o=self.port, i=self.port),
            '-v', '{o}:{i}'.format(o=config, i="/etc/docker/registry/config.yml"),
            REGISTRY_IMAGE]
        exit_code = subprocess.call(run_cmd)
        assert exit_code == 0
        while True:
            time.sleep(2)
            r = requests.get('http://{}/v2/'.format(self.addr))
            r.close()
            if r.status_code == 200:
                break
        print 'Started container ' + self.name

    def stop(self):
        print 'Stopping container ' + self.name
        subprocess.call(['docker', 'stop', self.name])
        print 'Removing docker container ' + self.name
        subprocess.call(['docker', 'rm', self.name])

    @property
    def addr(self):
        return "127.0.0.1:{}".format(str(self.port))


@pytest.fixture(scope='session')
def registry1():
    r = Registry(REGISTRY1_PORT, REGISTRY1_CONTAINER, REGISTRY1_CONFIG_PATH)
    yield r
    r.stop()


@pytest.fixture(scope='session')
def registry2():
    r = Registry(REGISTRY2_PORT, REGISTRY2_CONTAINER, REGISTRY2_CONFIG_PATH)
    yield r
    r.stop()


@pytest.fixture
def storage_dir():
    path = tempfile.mkdtemp(dir='/tmp')
    yield path
    try:
        shutil.rmtree(path)
    except Exception as e:
        print 'Cleanup error for {path}: {err}'.format(path=path, err=str(e))


@pytest.fixture
def cache_dir():
    path = tempfile.mkdtemp(dir='/tmp')
    yield path
    try:
        shutil.rmtree(path)
    except Exception as e:
        print 'Cleanup error for {path}: {err}'.format(path=path, err=str(e))
