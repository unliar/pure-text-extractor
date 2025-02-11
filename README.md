# pure-text-rss

# params

## process-rss

- `url`: RSS URL
- `separator`: item line separator default to `\n\n`
- `stripHTML`: whether to strip HTML tags default to `true`
- `length`: item length default to `0` (no limit)

## process-html

- `url`: HTML URL
- `separator`: item line separator default to `\n\n`
- `stripHTML`: whether to strip HTML tags default to `true`
- `selector`: CSS selector for the content, default to `body`. the detail can be found at https://github.com/PuerkitoBio/goquery

# example

`shell
curl "http://localhost:8080/process-rss?url=https://rsshub.flyneko.com/weibo/user/1888981347&separator=\n\n&stripHTML=false"

curl "http://localhost:8080/process-html?url=https://rsshub.flyneko.com&separator=\n\n&stripHTML=false&selector=body"
`

# docker deployment

## docker-compose

```yaml
version: "3"

services:
  - name: pure-text-extractor
    image: ghcr.io/unliar/pure-text-extractor:latest
    ports:
      - 8080:8080
    restart: always
```

## docker

```shell
docker run -d --name pure-text-extractor -p 8080:8080 ghcr.io/unliar/pure-text-extractor:latest
```
