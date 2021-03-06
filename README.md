# paramchecker
A more customizable and socks5 proxy-aware version of https://github.com/tomnomnom/hacks/blob/master/kxss.

## Usage

```shell
cat file_with_urls.txt | paramchecker [flags]

Flags:
  -worker    amount of worker running concurrently
  -proxy     socks5 proxy string socks5://<ip>:<port>
  -header    custom headers (can be used multiple times)
  -rate      requests per second
```
Be aware, setting the rate flag combined with a worker count > 1 will not keep the actual rates per second ratio.
Instead it can be said that **rate * worker = rates per seconds**.

## Upcoming
- Scanning multiple hosts asynchronously
- WAF detection and evasion
