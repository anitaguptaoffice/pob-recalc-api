# scripts

脚本依赖：

- 国服PathOfExile游戏文件
- 国际互联网访问能力（从github下载`schema.min.json`文件时可能需要）

脚本还依赖以下程序：

- Python 脚本执行环境
- Node.js pathofexile-dat的执行环境
- [uv](https://github.com/astral-sh/uv) 管理Python依赖
- [pathofexile-dat](https://github.com/PoEMBASite/poe-dat-viewer/blob/master/lib/README-CLI.md) 解包工具

## 配置

需要修改以下配置文件以满足实际情况：

- `export/game/global/config.json`，修改`patch`属性为最新的版本，可以在`https://snosme.github.io/poe-dat-viewer/`读取
- `export/game/tencent/config.json`，修改`steam`属性为国服安装路径
- `config.py`，修改`POB_PATH`为POB安装路径

## 使用

```shell
cd scripts
# 下载文件和解包
uv run ./main.py prepare
# 更新
uv run ./main.py
# 生成
uv run ./main.py make
```
