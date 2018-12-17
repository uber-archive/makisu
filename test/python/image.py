import json
import os.path
import shutil
import subprocess
import tarfile
import tempfile


class DockerImage():

    def __init__(self, image_name):
        self.image_name = image_name
        self.directory = tempfile.mkdtemp()
        self.image_tar = os.path.join(self.directory, 'image.tar')
        self._do_save()
        self._extract()
        self._load_manifest()

    def _do_save(self):
        command = ['docker', 'save', '-o', self.image_tar, self.image_name]
        exit_code = subprocess.call(command)
        if exit_code != 0:
            msg = 'Failed to save docker image to local filesystem: {} => {}'.format(
                self.image_name, self.image_tar)
            raise Exception(msg)

    def _extract(self):
        command = ['tar', '-xvf', self.image_tar, '-C', self.directory]
        exit_code = subprocess.call(command)
        if exit_code != 0:
            msg = 'Failed to extract docker image to local filesystem: {}'.format(
                self.image_name)
            raise Exception(msg)

    def _load_manifest(self):
        manifest_path = os.path.join(self.directory, 'manifest.json')
        fh = open(manifest_path, 'r')
        data = json.load(fh)
        self.manifest = data[0]

    def get_layer_locations(self):
        return self.manifest.get('Layers')

    def get_layer_by_offset(self, offset):
        layers = self.get_layer_locations()
        if offset >= len(layers):
            return None
        layer_location = layers[offset]
        layer_location = os.path.join(self.directory, layer_location)
        return DockerLayer(layer_location)

    def cleanup(self):
        shutil.rmtree(self.directory)


class DockerLayer():

    def __init__(self, layer_tar):
        self.layer_tar = layer_tar

    def get_tar_headers(self):
        tar = tarfile.open(self.layer_tar)
        for member in tar.getmembers():
            yield member

    def get_tar_header_count(self):
        try:
            return len(list(self.get_tar_headers()))
        except tarfile.ReadError:
            return 0

    def get_tar_header_by_name(self, name):
        for member in self.get_tar_headers():
            if member.name == name:
                return member
        return None
