import os
from pathlib import Path


USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"

POB_PATH = "D:\\AppsInDisk\\PoeCharm2-20260309\\PathOfBuildingCommunity"

DATA_ROOT = Path(os.path.dirname(os.path.abspath(__file__))).parent
PROJECT_ROOT = DATA_ROOT.parent
