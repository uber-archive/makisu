import os
import shutil

TEST_DIR = os.path.join(os.getcwd(), "integration/python/.tmp")
TEST_ROOT = "/tmp/makisu-test-integration"

if not os.path.exists(TEST_DIR):
    os.makedirs(TEST_DIR)

if os.path.exists(TEST_ROOT):
    shutil.rmtree(TEST_ROOT)
