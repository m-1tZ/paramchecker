# paramchecker
A more customizable and socks5 proxy-aware version of https://github.com/tomnomnom/hacks/blob/master/kxss.

## Usage

```shell
Flags:
  - worker    amount of worker running concurrently
  - proxy     socks5 proxy string socks5://<ip>:<port>
  - headers   custom headers separated by ;
  - rate      requests per second
```
Be aware, setting the rate flag combined with a worker count > 1 will not keep the actual rates per second ratio.
Instead it can be said that **rate * worker = rates per seconds**.