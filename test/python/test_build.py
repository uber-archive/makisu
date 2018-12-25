import os
import random
import subprocess
import tempfile
import utils

import image


def new_image_name():
    return "makisu-test:{}".format(random.randint(0, 1000000))


def test_build_simple(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/simple')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, load=True,
                             registry_config={"*": {"*": {"security": {"tls": {"client": {"disabled": True}}}}}})
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_numbered_alias(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/numbered-alias')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, load=True)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_copy_from_image(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/copy-from-image')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, load=True)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_delete_intermediate(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/delete-intermediate')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_symlink(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/symlink')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_with_readonly_mnt(registry1, storage_dir, cache_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/simple')

    _, test_file = tempfile.mkstemp(dir='/tmp')  # TODO: prevent leaking if test failed
    additional_volumes = {test_file: '/var/run/makisu/builder/test'}

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, cache_dir, additional_volumes)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_copyroot(registry1, storage_dir, cache_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/copyroot')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, cache_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_copy_glob(registry1, storage_dir, cache_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/copy-glob')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, cache_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_cache_copyfrom(registry1, storage_dir, cache_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/cache-copy-from')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, cache_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, cache_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_go_from_scratch(registry1, storage_dir):
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/go-from-scratch')

    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err


def test_build_commit_empty_pair(registry1, storage_dir, cache_dir, tmpdir):
    utils.registry_ensure_image('debian:8', registry1.addr)
    new_image1 = new_image_name()
    new_image2 = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/commit-empty-pair')
    test_file = tmpdir.join("f1")
    test_file.write("")

    # First build, mount in test file.
    additional_volumes = {test_file: '/mnt/f1'}
    utils.makisu_build_image(new_image1, registry1.addr, context_dir, storage_dir, cache_dir, additional_volumes)

    # Second build, without test file. It would fail if distributed cache doesn't work.
    utils.makisu_build_image(new_image2, registry1.addr, context_dir, storage_dir, cache_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image2)
    assert code == 0, err


def test_build_with_cache(registry1, storage_dir, cache_dir, tmpdir):
    utils.registry_ensure_image('debian:8', registry1.addr)
    new_image1 = new_image_name()
    new_image2 = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/mount')
    test_file = tmpdir.join("f1")
    test_file.write("")
    test_file2 = tmpdir.join("f2")
    test_file2.write("")

    # First build, mount in test file.
    additional_volumes = {test_file: '/tmp/test.txt', test_file2: '/root/mounted.txt'}
    utils.makisu_build_image(new_image1, registry1.addr, context_dir, storage_dir, cache_dir, additional_volumes)

    # Second build, without test file. It would fail if distributed cache doesn't work.
    utils.makisu_build_image(new_image2, registry1.addr, context_dir, storage_dir, cache_dir)
    code, err = utils.docker_run_image(registry1.addr, new_image2)
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
    base_layers = image.DockerImage("127.0.0.1:5002/debian:8").get_layer_locations()

    img = image.DockerImage(image_name)
    assert len(img.get_layer_locations()) == len(base_layers) + 1

    l1 = img.get_layer_by_offset(1)
    assert l1.get_tar_header_count() == 1, [h.name for h in l1.get_tar_headers()]
    assert list(l1.get_tar_headers())[0].uname != "root"
    assert list(l1.get_tar_headers())[0].gname != "root"

    img.cleanup()


def test_build_go_with_debian_package(registry1, storage_dir):
    utils.registry_ensure_image('golang:latest', registry1.addr)
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/go-with-debian-package')

    docker_build_args = [
        "BASE_IMAGE=127.0.0.1:5002/golang:latest",
    ]
    utils.makisu_build_image(new_image, registry1.addr, context_dir, storage_dir, docker_args=docker_build_args)
    utils.docker_pull_image('{}/{}'.format(registry1.addr, new_image))

    proc = subprocess.Popen([
        "docker", "run", "-i", "--rm", '{}/{}'.format(registry1.addr, new_image),
    ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    output, err = proc.communicate()
    assert proc.returncode == 0, err
    assert output.find("/usr/bin/hello") != -1

    image_name = '{}/{}'.format(registry1.addr, new_image)
    base_layers = image.DockerImage("127.0.0.1:5002/golang:latest").get_layer_locations()

    img = image.DockerImage(image_name)
    assert len(img.get_layer_locations()) == len(base_layers) + 2

    l2 = img.get_layer_by_offset(-1)
    assert l2.get_tar_header_count() == 1, [h.name for h in l2.get_tar_headers()]
    assert list(l2.get_tar_headers())[0].uname != "root"
    assert list(l2.get_tar_headers())[0].gname != "root"

    img.cleanup()


def test_build_arg_and_env(registry1, storage_dir):
    utils.registry_ensure_image('golang:latest', registry1.addr)
    utils.registry_ensure_image('alpine:latest', registry1.addr)
    new_image = new_image_name()
    context_dir = os.path.join(os.getcwd(), 'testdata/build-context/arg-and-env')

    docker_build_args = [
        "BASE_IMAGE=127.0.0.1:5002/golang:latest",
        "RUNTIME_BASE_IMAGE=127.0.0.1:5002/alpine:latest",
    ]
    utils.makisu_build_image(
        new_image, registry1.addr, context_dir, storage_dir,
        docker_args=docker_build_args)
    utils.docker_pull_image('{}/{}'.format(registry1.addr, new_image))

    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err
