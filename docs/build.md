```shell
lsof -t -i:2048 | xargs kill || true && make build && ./dist/dbm > server.log 2>&1 &
```