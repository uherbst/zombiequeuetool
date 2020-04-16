# zombiequeuetool - list unused queues on Solace brokers

## Disclaimer

zombiequeuetool is not developed and maintained by Solace.<br/>
It is currently only tested against PubSub+ software brokers (VMRs), not appliances. But most probably, it will works with appliances without any issues.<br/>

## Features
zombiequeuetool is written in go, based on the Solace Legacy SEMP protocol.<br/>

Every second, get a list of queues, that have no consumer bound to them. Repeat that for the given duration.

All queues, that have never bound a consumer to them in that time, are meant as "unused" and are listed at STDOUT, one queue per line.

## Usage
<pre><code>zombiequeuetool  -h
Usage of ./zombiequeuetool:
  -debug
        Enable debug mode
  -duration int
        how long to wait for queues without binding ? (default 10)
 -filter string
        Regex-filter for msg-vpn/queue-names
  -insecure
        do not verify TLS server certificate
  -password string
        password for the SEMP access
  -url string
        URL to the SEMP service
  -user string
        username for the SEMP access
</code></pre>

## Output formt

To uniqly identify a queue, you need also the name of the message-vpn - yes, you can have the same queue name in multiple message-vpns.

The output format is:
msg-vpn-name|queue-name

## Filtering the output
All regexes supported from the go-regexp-package can be used to filter the output - for both parts: msg-vpn-name and queue name.

To be short: have a look at

```
go doc regexp/syntax
```

to see a full description on supported regexes.

### Regex examples

```
zombiequeuetool -filter 'q.2'

Outputs:
testvpn|q12
testvpn|q72

but not:
testvpn2|q1


