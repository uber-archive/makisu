import os
import subprocess
import tempfile

from .utils import new_image_name, docker_save_image, makisu_push_image, docker_run_image


def test_push_simple(registry1, registry2, storage_dir, tmpdir):
    new_image = new_image_name()
    replica_image = new_image_name()

    _, image_tar_path = tempfile.mkstemp(dir='/tmp')  # TODO: prevent leaking if test failed.
    docker_save_image('busybox:latest', image_tar_path)

    makisu_push_image(
        new_image, image_tar_path,
        registry=registry1.addr,
        replicas=[registry2.addr + '/' + replica_image],
        registry_cfg={"*": {"*": {"security": {"tls": {"client": {"disabled": True}}}}}})
    code, err = docker_run_image(registry1.addr, new_image)
    assert code == 0, err
    code, err = docker_run_image(registry2.addr, replica_image)
    assert code == 0, err

