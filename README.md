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
  -insecure
        do not verify TLS server certificate
  -password string
        password for the SEMP access
  -url string
        URL to the SEMP service
  -user string
        username for the SEMP access
</code>

