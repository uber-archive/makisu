import os
import subprocess
import tempfile
import utils

import image


def test_push_simple(registry1, registry2, storage_dir, tmpdir):
    new_image = utils.new_image_name()
    replica_image = utils.new_image_name()

    _, image_tar_path = tempfile.mkstemp(dir='/tmp')  # TODO: prevent leaking if test failed.
    utils.docker_save_image('busybox:latest', image_tar_path)

    utils.makisu_push_image(
        new_image, image_tar_path,
        registry=registry1.addr,
        replicas=[registry2.addr + '/' + replica_image],
        registry_cfg={"*": {"*": {"security": {"tls": {"client": {"disabled": True}}}}}})
    code, err = utils.docker_run_image(registry1.addr, new_image)
    assert code == 0, err
    code, err = utils.docker_run_image(registry2.addr, replica_image)
    assert code == 0, err

