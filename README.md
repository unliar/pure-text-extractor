# pure-text-rss

# params

- `url`: RSS URL
- `separator`: item line separator default to `\n\n`
- `stripHTML`: whether to strip HTML tags default to `true`
- `length`: item length default to `0` (no limit)

# example

`shell
curl "http://localhost:8080/process-rss?url=https://rsshub.flyneko.com/weibo/user/1888981347&separator=\n\n&stripHTML=false"
`

# docker deployment

## docker-compose

```yaml
version: "3"

services:
  - name: pure-text-rss
    image: registry.cn-shenzhen.aliyuncs.com/unliar/pure-text-rss:latest
    ports:
      - 8080:8080
    restart: always
```

## docker

```shell
docker run -d --name pure-text-rss -p 8080:8080 registry.cn-shenzhen.aliyuncs.com/unliar/pure-text-rss:latest
```
