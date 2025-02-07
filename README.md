# pure-text-rss

# params

- `url`: RSS URL
- `separator`: item line separator default to `\n\n`
- `stripHTML`: whether to strip HTML tags default to `true`

# example

`shell
curl "http://localhost:8080/process-rss?url=https://rsshub.flyneko.com/weibo/user/1888981347&separator=\n\n&stripHTML=false"
`
