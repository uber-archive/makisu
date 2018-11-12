package storage

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/uber/makisu/lib/utils/testutil"
)

// StoreFixture returns test store.
func StoreFixture() (ImageStore, func()) {
	root, err := ioutil.TempDir("/tmp", "makisu-test")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := recover(); err != nil {
			os.RemoveAll(root)
			log.Fatal(err)
		}
	}()

	store, err := NewImageStore(root)
	if err != nil {
		log.Fatal(err)
	}
	return store, func() {
		os.RemoveAll(root)
	}
}

// StoreFixtureWithSampleImage returns test store with sample image.
func StoreFixtureWithSampleImage() (ImageStore, func()) {
	store, c := StoreFixture()

	manifestData, err := ioutil.ReadFile("../../testdata/files/test_distribution_manifest")
	if err != nil {
		log.Fatal(err)
	}
	if err := store.Manifests.CreateDownloadFile(
		testutil.SampleImageRepoName, testutil.SampleImageTag, 0); err != nil {
		log.Fatal(err)
	}
	w, err := store.Manifests.GetDownloadFileReadWriter(
		testutil.SampleImageRepoName, testutil.SampleImageTag)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(manifestData)
	w.Close()
	if err := store.Manifests.MoveDownloadFileToStore(
		testutil.SampleImageRepoName, testutil.SampleImageTag); err != nil {
		log.Fatal(err)
	}

	imageConfigData, err := ioutil.ReadFile("../../testdata/files/test_image_config")
	if err != nil {
		log.Fatal(err)
	}
	if err := store.Layers.CreateDownloadFile(testutil.SampleImageConfigDigest, 0); err != nil {
		log.Fatal(err)
	}
	w, err = store.Layers.GetDownloadFileReadWriter(testutil.SampleImageConfigDigest)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(imageConfigData)
	w.Close()
	if err := store.Layers.MoveDownloadFileToStore(testutil.SampleImageConfigDigest); err != nil {
		log.Fatal(err)
	}

	layerTarData, err := ioutil.ReadFile("../../testdata/files/test_layer.tar")
	if err != nil {
		log.Fatal(err)
	}
	if err := store.Layers.CreateDownloadFile(testutil.SampleLayerTarDigest, 0); err != nil {
		log.Fatal(err)
	}
	w, err = store.Layers.GetDownloadFileReadWriter(testutil.SampleLayerTarDigest)
	if err != nil {
		log.Fatal(err)
	}
	w.Write(layerTarData)
	w.Close()
	if err := store.Layers.MoveDownloadFileToStore(testutil.SampleLayerTarDigest); err != nil {
		log.Fatal(err)
	}

	return store, c
}
