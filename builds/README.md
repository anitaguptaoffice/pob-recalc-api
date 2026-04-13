# builds/

存放 POB build 数据文件，用于本地测试和调试。

## 文件格式

- `.txt` — POB 分享码（base64url + zlib 压缩的 XML）
- `.xml` — 解码后的 POB XML

## 使用方式

```bash
# 测试 recalc
curl -X POST http://localhost:8080/recalc --data-binary @builds/my_build.txt

# 测试造价
curl -X POST http://localhost:8080/build-cost \
  -H 'Content-Type: application/json' \
  -d '{"pob_code": "'"$(cat builds/my_build.txt)"'", "poesessid": "xxx", "cn_league": "S29赛季"}'
```

## 注意

此目录已被 `.gitignore` 忽略，个人 build 数据不会提交到仓库。
测试用的 fixture 数据在 `tests/testdata/` 中。
