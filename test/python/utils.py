import json
import os.path
import requests
import string
import subprocess


def docker_image_exists(image):
    output = subprocess.check_output([
        'docker', 'images', image, '--format', '"{{.Repository}}:{{.Tag}}"'
    ])
    return image in output


def docker_pull_image(image):
    if not docker_image_exists(image):
        exit_code = subprocess.call(['docker', 'pull', image])
        assert exit_code == 0
    assert docker_image_exists(image)


def docker_save_image(image, tar_path):
    docker_pull_image(image)
    exit_code = subprocess.call(['docker', 'save', image, '-o', tar_path])
    assert exit_code == 0


def docker_tag_image(old_image, new_image):
    docker_pull_image(old_image)
    exit_code = subprocess.call(['docker', 'tag', old_image, new_image])
    assert exit_code == 0
    assert docker_image_exists(new_image)


def docker_push_image(image, registry):
    if not registry_image_exists(image, registry):
        tagged_image = '{r}/{i}'.format(r=registry, i=image)
        docker_tag_image(image, tagged_image)
        exit_code = subprocess.call(['docker', 'push', tagged_image])
        assert exit_code == 0
    assert registry_image_exists(image, registry)


def docker_delete_image(image):
    if docker_image_exists(image):
        exit_code = subprocess.call(['docker', 'rmi', image])
        assert exit_code == 0
    assert not docker_image_exists(image)


def registry_image_exists(image, registry):
    repotag = string.split(image, ':')
    assert len(repotag) >= 2
    tag = repotag[-1]
    image = string.lstrip("".join(repotag[:-1]), registry)
    url = os.path.join('http://' + registry, 'v2', image, 'manifests', tag)
    r = requests.get(url)
    r.close()
    return r.status_code == 200


def registry_ensure_image(image, registry):
    docker_pull_image(image)
    docker_push_image(image, registry)


def get_base_image():
    version = os.getenv("PACKAGE_VERSION", "latest")
    if os.getenv("MAKISU_ALPINE", "0") != "0":
        return "makisu-alpine:{}".format(version)
    return "makisu:{}".format(version)


def makisu_run_cmd(volumes, args):
    cmd = ['docker', 'run', '-i', '--rm', '--net', 'host']

    # Add volumes to docker command.
    volumes['/var/run/docker.sock'] = '/docker.sock'  # Mount docker socket
    for k, v in volumes.iteritems():
        cmd.extend([
            '-v',
            '{p_outside}:{p_inside}'.format(p_outside=k, p_inside=v),
        ])

    cmd.extend([
        '-e', 'DOCKER_HOST=unix:///docker.sock',
    ])

    # Set entrypoint and image to be launched through docker command.
    cmd.append(get_base_image())
    cmd.extend(args)

    print 'Running docker command: ', ' '.join(cmd)

    exit_code = subprocess.call(cmd)
    return exit_code


def makisu_build_image(new_image, registry, context_dir, storage_dir,
                       cache_dir=None, additional_volumes=None,
                       docker_args=None, load=False, registry_config=None):
    volumes = additional_volumes or {}
    volumes[storage_dir] = storage_dir  # Sandbox and file store
    volumes[context_dir] = '/context'  # Mount context dir
    if cache_dir:
        volumes[cache_dir] = cache_dir  # Cache key-value store

    docker_args = docker_args or {}

    args = [
        'build',
        '-t', '{}'.format(new_image),
        '--storage', storage_dir,
        '--push', registry,
        '--modifyfs=true',
        '--commit=explicit',
    ]
    for docker_arg in docker_args:
        args.extend(['--build-arg', docker_arg])

    if registry_config is not None:
        args.extend(['--registry-config', json.dumps(registry_config)])

    if load:
        args.append('--load')

    if not cache_dir:
        args.extend(['--local-cache-ttl', '0s'])

    args.append('/context')

    exit_code = makisu_run_cmd(volumes, args)
    assert exit_code == 0
    assert registry_image_exists(new_image, registry)


def docker_run_image(registry, image):
    proc = subprocess.Popen([
        "docker", "run", "-i", "--rm",
        '{}/{}'.format(registry, image),
    ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    _, err = proc.communicate()
    return proc.returncode, err
