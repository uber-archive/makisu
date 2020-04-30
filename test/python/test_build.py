import os
import subprocess
import tempfile

from .utils import docker_pull_image, docker_push_image, docker_run_image, get_base_image, makisu_build_image, new_image_name
from .image import DockerImage

def test_build_simple(registry1, registry2, storage_dir):
    new_image = new_image_name()
    replica_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/simple')

    makisu_build_image(
        new_image, context_dir, storage_dir, load=True, registry=registry1.addr,
        replicas=[registry2.addr + "/" + replica_image],
        registry_cfg={"*": {"*": {"security": {"tls": {"client": {"disabled": True}}}}}})
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err
    code, err = docker_run_image(registry2.addr, replica_image)
    assert code == 0, err


def test_build_remove(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/remove')
    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_from_sha256(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/from-sha256')
    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_symlink(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/symlink')

    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_copy_glob(registry1, storage_dir, cache_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/copy-glob')

    makisu_build_image(
        new_image, context_dir, storage_dir, cache_dir=cache_dir,
        registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_copy_from(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/copy-from')

    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_copy_add_chown(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/copy-add-chown')

    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_arg_and_env(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/arg-and-env')

    docker_build_args = [
        "BASE_IMAGE=golang:latest",
        "RUNTIME_BASE_IMAGE=alpine:latest",
    ]
    makisu_build_image(
        new_image, context_dir, storage_dir, docker_args=docker_build_args,
        registry=registry1.addr)
    docker_pull_image('{}/{}'.format(registry1.addr, new_image))

    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_user_change(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/user-change')

    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


# When we run makisu build, some mountpoints and their parent directories are
# already created. As such, when we unpack `/var/run -> /run` symlink in
# `debian:9` for instance, we get an error because we think we should delete
# everything under `/var/run`, but the mountpoint might be read-only, thus
# returning an error.
# We fix that by checking if `/var/run` is present and contains a mountpoint
# right before a build. If that is the case then we choose to ignore all file
# changes under that directory.
# This is necessary in the k8s environment, because the default token is mounted
# under `/var/run/secrets/kubernetes/serviceaccount`.
def test_build_with_readonly_var_run_mnt(registry1, storage_dir, cache_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/simple')

    _, test_file = tempfile.mkstemp(dir='/tmp')  # TODO: prevent leaking if test failed
    volumes = {test_file: '/var/run/makisu/builder/test'}

    makisu_build_image(
        new_image, context_dir, storage_dir, cache_dir=cache_dir,
        volumes=volumes, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_go_from_scratch(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/go-from-scratch')

    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_with_distributed_cache(registry1, storage_dir, cache_dir, tmpdir):
    new_image1 = new_image_name()
    new_image2 = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/mount')
    test_file = tmpdir.join("f1")
    test_file.write("")
    test_file2 = tmpdir.join("f2")
    test_file2.write("")

    # First build, mount in test file.
    volumes = {test_file: '/tmp/test.txt', test_file2: '/root/mounted.txt'}
    makisu_build_image(
        new_image1, context_dir, storage_dir, cache_dir=cache_dir,
        volumes=volumes, registry=registry1.addr)

    # Second build, without test file.
    # It would fail if distributed cache doesn't work.
    makisu_build_image(
        new_image2, context_dir, storage_dir, cache_dir=cache_dir,
        registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image2)
    assert code == 0, err

    proc = subprocess.Popen([
        "docker", "run", "-i", "--rm",
        '--entrypoint', '/bin/bash -c cat /root/mounted.txt',
        '{}/{}'.format(registry1.addr, new_image2),
    ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    _, err = proc.communicate()
    assert proc.returncode != 0, err
    assert err != ""

    image_name = '{}/{}'.format(registry1.addr, new_image2)
    base_layers = DockerImage("alpine:latest").get_layer_locations()

    img = DockerImage(image_name)
    assert len(img.get_layer_locations()) == len(base_layers) + 3

    l1 = img.get_layer_by_offset(3)
    assert l1.get_tar_header_count() == 1, [h.name for h in l1.get_tar_headers()]
    assert list(l1.get_tar_headers())[0].uname != "root"
    assert list(l1.get_tar_headers())[0].gname != "root"


def test_build_with_local_cache(registry1, storage_dir, cache_dir, tmpdir):
    new_image1 = new_image_name()
    new_image2 = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/mount')
    test_file = tmpdir.join("f1")
    test_file.write("")
    test_file2 = tmpdir.join("f2")
    test_file2.write("")

    # First build, mount in test file.
    volumes = {test_file: '/tmp/test.txt', test_file2: '/root/mounted.txt'}
    makisu_build_image(
        new_image1, context_dir, storage_dir, cache_dir=cache_dir,
        volumes=volumes, registry=registry1.addr)


    # Second build, without test file and without registry config.
    # It would fail if local cache doesn't work.
    makisu_build_image(
        new_image2, context_dir, storage_dir, cache_dir=cache_dir,
        volumes=volumes, load=True)

    code, err = docker_run_image(None, new_image2)
    assert code == 0, err


def test_build_go_with_debian_package(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/go-with-debian-package')

    docker_build_args = [
        "BASE_IMAGE=golang:latest",
    ]
    makisu_build_image(
        new_image, context_dir, storage_dir, docker_args=docker_build_args,
        registry=registry1.addr)
    docker_pull_image('{}/{}'.format(registry1.addr, new_image))

    proc = subprocess.Popen([
        "docker", "run", "-i", "--rm", '{}/{}'.format(registry1.addr, new_image),
    ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output, err = proc.communicate()
    assert proc.returncode == 0, err
    assert output.find("/usr/bin/hello") != -1

    image_name = '{}/{}'.format(registry1.addr, new_image)
    base_layers = DockerImage("golang:latest").get_layer_locations()

    img = DockerImage(image_name)
    assert len(img.get_layer_locations()) == len(base_layers) + 2

    l2 = img.get_layer_by_offset(-1)
    assert l2.get_tar_header_count() == 1, [h.name for h in l2.get_tar_headers()]
    assert list(l2.get_tar_headers())[0].uname != "root"
    assert list(l2.get_tar_headers())[0].gname != "root"


# A little bit of crazyness on this test but I didn't want to add more complex
# methods to test the `--preserve-root` flag.
# See the [Dockerfile](../testdata/build-context/preserve-root/Dockerfile)
def test_build_with_preserve_root(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/preserve-root')

    os.environ["MAKISU_ALPINE"] = "1"

    docker_push_image(get_base_image(), registry1.addr)

    docker_build_args = [
        "BASE_IMAGE={}/{}".format(registry1.addr, get_base_image()),
    ]
    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr,
        docker_args=docker_build_args, load=True)
    code, err = docker_run_image(registry1.addr, new_image)

    del os.environ["MAKISU_ALPINE"]

    assert code == 0, err

def test_build_entrypoint(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/entrypoint')
    makisu_build_image(
        new_image, context_dir, storage_dir, registry=registry1.addr)
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err

def test_build_global_arg(registry1, storage_dir):
    new_image = utils.new_image_name()
    context_dir = os.path.join(
        os.getcwd(), 'testdata/build-context/global-arg')
    docker_build_args = [
        "version_default=v2",
    ]
    utils.makisu_build_image(
        new_image, context_dir, storage_dir,
        registry=registry1.addr, docker_args=docker_build_args)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err
