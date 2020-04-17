// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Arguments struct {
	url, user, password *string
	duration            *int
	insecure            *bool
	debug               *bool
	filter              *string
	delete              *bool
}

const separator = "@" // separates msg-vpn-name and queue name

// parseCommandLine parses cmd line arguments and puts them into Arguments struct.
func doCommandLine() Arguments {
	var a Arguments

	a.url = flag.String("url", "", "URL to the SEMP service")
	a.user = flag.String("user", "", "username for the SEMP access")
	a.password = flag.String("password", "", "password for the SEMP access")
	a.duration = flag.Int("duration", 10, "how long to wait for queues without binding ?")
	a.debug = flag.Bool("debug", false, "Enable debug mode")
	a.insecure = flag.Bool("insecure", false, "do not verify TLS server certificate")
	a.filter = flag.String("filter", "", "Regex-filter for msg-vpn/queue-names")
	a.delete = flag.Bool("delete", false, "delete unused queues")
	flag.Parse()

	if len(*a.password) == 0 {
		fmt.Println("Password not set!")
	}

	if *a.debug {
		fmt.Println("------------ command line values ------------")
		fmt.Println("URL: ", *a.url)
		fmt.Println("Username: ", *a.user)
		fmt.Println("Password: xxxx")
		fmt.Println("Duration: ", *a.duration)
		fmt.Println("Debugmode: ", *a.debug)
		fmt.Println("dont validate TLS: ", *a.insecure)
		fmt.Println("Regex Filter: ", *a.filter)
		fmt.Println("Delete: ", *a.delete)
		fmt.Println("------------ End of command line values ------")
		fmt.Println()
	}
	return a
}

// listQueuesWithoutConsumer list all queues on a Solace broker without consumer bound to them
func listQueuesWithoutConsumer(a Arguments) (map[string]bool, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName string  `xml:"message-vpn"`
								BindCount  float64 `xml:"bind-count"`
							} `xml:"info"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	queues := make(map[string]bool)
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: *a.insecure}}
	client := http.Client{
		Transport: tr,
	}
	for command := "<rpc><show><queue><name>*</name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; command != ""; {

		req, err := http.NewRequest("GET", *a.url+"/SEMP", strings.NewReader(command))
		if err != nil {
			log.Println("There is an error in net/http/NewRequest: ", err)
			return nil, err
		}
		req.SetBasicAuth(*a.user, *a.password)
		req.Header.Set("Content-type", "application/xml")

		resp, err := client.Do(req)

		if err != nil {
			log.Println("There is an error in net/http/client.Do: ", err)
			return nil, err
		}
		defer resp.Body.Close()
		decoder := xml.NewDecoder(resp.Body)
		var queueList Data
		err = decoder.Decode(&queueList)
		if err != nil {
			log.Println("There is an error in decoding xml: ", err)
			return nil, err
		}
		if queueList.ExecuteResult.Result != "ok" {
			log.Println("There is an error in decoding xml: ", queueList.ExecuteResult.Result)
			return nil, err
		}

		command = queueList.MoreCookie.RPC

		for _, queue := range queueList.RPC.Show.Queue.Queues.Queue {
			if queue.Info.BindCount == 0 {
				queues[queue.Info.MsgVpnName+separator+queue.QueueName] = true
			}
		}
	}

	return queues, nil
}
func deleteQueue(a Arguments, q string) error {
	regex := regexp.MustCompile("(.*)" + separator + "(.*)")
	names := regex.FindStringSubmatch(q)

	msgvpn := names[1]
	queue := names[2]

	type Data struct {
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: *a.insecure}}
	client := http.Client{
		Transport: tr,
	}

	command := "<rpc><message-spool><vpn-name>" + msgvpn +
		"</vpn-name><no><queue><name>" + queue +
		"</name></queue></no></message-spool></rpc>"

	req, err := http.NewRequest("GET", *a.url+"/SEMP", strings.NewReader(command))
	if err != nil {
		log.Println("There is an error in net/http/NewRequest: ", err)
		return err
	}
	req.SetBasicAuth(*a.user, *a.password)
	req.Header.Set("Content-type", "application/xml")

	resp, err := client.Do(req)

	if err != nil {
		log.Println("There is an error in net/http/client.Do: ", err)
		return err
	}
	defer resp.Body.Close()
	decoder := xml.NewDecoder(resp.Body)
	var result Data
	err = decoder.Decode(&result)
	if err != nil {
		log.Println("There is an error in decoding xml: ", err)
		return err
	}
	if result.ExecuteResult.Result != "ok" {
		log.Println("There is an error in deleting queue: ", result.ExecuteResult.Result)
		return err
	}
	if *a.debug {
		fmt.Println("Result of deleting queue" + q + ": " + result.ExecuteResult.Result)
	}
	return nil
}

func main() {
	// Handling command line
	cmdargs := doCommandLine()
	startTime := time.Now()
	queues, err := listQueuesWithoutConsumer(cmdargs)
	if err != nil {
		log.Println("There is an error in listQueuesWithoutConsumer ", err)
		return
	}

	// conditions for that for loop:
	// time.Now().sub(startTime).Seconds(): Which time is elapsed since startTime (in seconds)?
	// len(queues) > 0: If there are no queues without bind, we can stop everything
	for time.Now().Sub(startTime).Seconds() < float64(*cmdargs.duration) && len(queues) > 0 {
		newQueues, err := listQueuesWithoutConsumer(cmdargs)
		if err != nil {
			log.Println("There is an error in listQueuesWithoutConsumer ", err)
			return
		}

		// Loop over all queues listed in queues: Still without bind in newQueues ?
		for currentQueue, _ := range queues {
			if !newQueues[currentQueue] {
				// if not listed in newQueues, this queue HAS binds or does not exist anymore.
				// in both cases: We dont care about that queue anymore.
				if *cmdargs.debug {
					fmt.Println("Queue " + currentQueue + " found with bound consumer.")
				}
				delete(queues, currentQueue)
			}
		}
		time.Sleep(1 * time.Second)
	}

	numberQueuesFound := 0
	regex := regexp.MustCompile(*cmdargs.filter)
	for queue, _ := range queues {
		if regex.MatchString(queue) {
			numberQueuesFound++
			fmt.Print(queue)
			if *cmdargs.delete {
				deleteQueue(cmdargs, queue)
				fmt.Print(" ... deleted")
			}
			fmt.Println()
		}
	}
	if numberQueuesFound == 0 {
		fmt.Println("No queues without binding found")
	}
}
