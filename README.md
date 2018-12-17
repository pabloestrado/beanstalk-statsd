# beanstalk-statsd

Send [beanstalk](http://kr.github.io/beanstalkd/) stats to [StatsD](https://github.com/etsy/statsd)

This is a fork of [beanstalk-statsd](https://github.com/RealGeeks/beanstalk-statsd)

Main advantage of this project in compare with original one that *all* numeric metrics (string metrics will be skiped) from beanstalk should be send to statsd, including the global system metrics, they will be send to statsd with tube name "_global"

## Usage of beanstalk-statsd:
./beanstalk-statsd
    -beanstalkd="127.0.0.1:11300": Beanstalkd address
    -period=1s: How often to send stats. Ex.: 1s (second), 2m (minutes), 400ms (milliseconds)
    -prefix="beanstalk": StatsD prefix for all stats
    -statsd="127.0.0.1:8125": StatsD server address
    -tubes="*": Comma separated list of tubes to watch. Use * to watch all
    -v=1: Output verbosity level. Use 0 (quiet), 1 or 2
    -version=false: Show version and exit

## Install

Build it with ```go build```` Tested with Go 1.10
