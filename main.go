// Command to collect stats from banstalkd tubes and send to statsd
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cactus/go-statsd-client/statsd"
	"github.com/kr/beanstalk"
)

var config struct {
	BeanstalkdAddr string
	StatsdAddr     string
	StatsdPrefix   string
	Verbosity      int
	Period         time.Duration
	Tubes          map[string]bool
}

const Version = "1.0"

func main() {

	var tubes string
	var showver bool

	flag.BoolVar(&showver, "version", false, "Show version and exit")
	flag.StringVar(&config.BeanstalkdAddr, "beanstalkd", "127.0.0.1:11300", "Beanstalkd address")
	flag.StringVar(&config.StatsdAddr, "statsd", "127.0.0.1:8125", "StatsD server address")
	flag.StringVar(&config.StatsdPrefix, "prefix", "beanstalk", "StatsD prefix for all stats")
	flag.IntVar(&config.Verbosity, "v", 1, "Output verbosity level. Use 0 (quiet), 1 or 2")
	flag.DurationVar(&config.Period, "period", time.Second, "How often to send stats. Ex.: 1s (second), 2m (minutes), 400ms (milliseconds)")
	flag.StringVar(&tubes, "tubes", "*", "Comma separated list of tubes to watch. Use * to watch all")
	flag.Parse()

	if showver {
		fmt.Fprintf(os.Stderr, "%s %s (%s)\n", os.Args[0], Version, runtime.Version())
		return
	}

	var err error
	config.Tubes, err = parseTubesWatch(tubes)
	if err != nil {
		log.Fatal(err)
	}

	for {
		stats, err := TubesStats()
		if err != nil {
			log.Print("ERROR (retry): ", err)
			continue
		}
		SendStats(stats)
		time.Sleep(config.Period)
	}
}

// TubesStats connects to beanstalkd and return a hash of all stats
// for each tube we are watching
//
// Return error if fail to connect to beanstalkd of fail to get stats
// for a specic tube
//
// Panic if a specic stat is not an integer, this should never happen
func TubesStats() (stats map[string]map[string]int, err error) {
	conn, err := beanstalk.Dial("tcp", config.BeanstalkdAddr)
	if err != nil {
		return stats, fmt.Errorf("Failed to connect to beanstalkd: %s", err)
	}
	tubes, err := conn.ListTubes()
	if err != nil {
		return stats, fmt.Errorf("Failed to list tubes: %s", err)
	}
	stats = map[string]map[string]int{}
	for _, tubeName := range tubes {
		if !watchingTube(tubeName) {
			continue
		}
		tube := &beanstalk.Tube{
			Name: tubeName,
			Conn: conn,
		}
		data, err := tube.Stats()
		if err != nil {
			return stats, fmt.Errorf("Failed to get stats for tube %s: %s", tubeName, err)
		}
		stats[tubeName] = make(map[string]int)
		for  k, v := range data {
			if vi, err := strconv.Atoi(v); err == nil {
 				stats[tubeName][k] = vi
			}			
		}
	}
	// Global stats	
	stats["_global"] = make(map[string]int)
	globalStats, err := conn.Stats()
	if err == nil {
		for metric, value := range globalStats {
			if vi, err := strconv.Atoi(value); err == nil {
				stats["_global"][metric] = vi
			}	
		}
	}


	return stats, nil
}

// SendStats will send all stats collected by TubesStats() to statsd
//
// Return error if fail to connect to statsd
func SendStats(stats map[string]map[string]int) error {
	client, err := statsd.NewClient(config.StatsdAddr, config.StatsdPrefix)
	if err != nil {
		return fmt.Errorf("Failed to connect to statsd: %s", err)
	}
	for tube, sts := range stats {
		verbose1("sending stats of tube %s", tube)
		for stat, value := range sts {
			name := fmt.Sprintf("%s.%s", tube, stat)
			client.Gauge(name, int64(value), 1.0)
			verbose2("%s.%s: %d", config.StatsdPrefix, name, value)
		}
	}
	return nil
}

// parseTubesWatch parses the -tubes command line argument into
// a hash 'tube name' -> true
//
// If we are watching all tubes, the hash will contain '*' -> true
func parseTubesWatch(tubes string) (map[string]bool, error) {
	tubes = strings.Trim(tubes, " ")
	if tubes == "" {
		return map[string]bool{}, errors.New("-tubes can't be blank")
	}
	if tubes == "*" {
		return map[string]bool{"*": true}, nil
	}
	t := map[string]bool{}
	for _, tube := range strings.Split(tubes, ",") {
		tube = strings.Trim(tube, " ")
		if tube != "" {
			t[tube] = true
		}
	}
	if len(t) == 0 {
		return map[string]bool{}, errors.New("-tubes can't be blank")
	}
	return t, nil
}

func watchingTube(name string) bool {
	if config.Tubes["*"] == true {
		return true
	}
	return config.Tubes[name] == true
}

func mustInt(n string) int {
	i, err := strconv.Atoi(n)
	if err != nil {
		log.Panic(err)
	}
	return i
}

func verbose1(format string, v ...interface{}) {
	if config.Verbosity >= 1 {
		log.Printf(format, v...)
	}
}

func verbose2(format string, v ...interface{}) {
	if config.Verbosity >= 2 {
		log.Printf(format, v...)
	}
}
