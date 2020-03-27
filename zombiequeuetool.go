package main

import (
	"crypto/tls"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Arguments struct {
	url, user, password *string
	duration            *int
	insecure            *bool
	debug               *bool
}

// parseCommandLine parses cmd line arguments and puts them into Arguments struct.
func doCommandLine() Arguments {
	var a Arguments

	a.url = flag.String("url", "", "URL to the SEMP service")
	a.user = flag.String("user", "", "username for the SEMP access")
	a.password = flag.String("password", "", "password for the SEMP access")
	a.duration = flag.Int("duration", 10, "how long to wait for queues without binding ?")
	a.debug = flag.Bool("debug", false, "Enable debug mode")
	a.insecure = flag.Bool("insecure", false, "do not verify TLS server certificate")
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
								MsgVpnName string `xml:"message-vpn"`
								// Quota           float64 `xml:"quota"`
								// Usage           float64 `xml:"current-spool-usage-in-mb"`
								// SpooledMsgCount float64 `xml:"num-messages-spooled"`
								BindCount float64 `xml:"bind-count"`
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
				queues[queue.QueueName] = true
			}
		}
	}

	return queues, nil
}

func main() {

	/*
		Test cases

		- SEMPv2 TLS, 1 queue
		- SEMPv2 TLS, viele Queues (mehr als in einen Aufruf passt)
		- SEMPv2 TLS, keine queue
		- SEMPv2, und nach 10 Sek. bekommt die Queue einen Bind
	*/

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
				delete(queues, currentQueue)
			}
		}
		/*
			Test cases
			- SEMPv2 plain, 1 queue
			- SEMPv2 TLS, 1 queue
			- SEMPv2 TLS, viele Queues (mehr als in einen Aufruf passt)
			- SEMPv2 TLS, keine queue
			- SEMPv2, und nach 10 Sek. bekommt die Queue einen Bind
		*/
		time.Sleep(1 * time.Second)
	}

	if len(queues) == 0 {
		if *cmdargs.debug {
			fmt.Println("No queues without binding found")
		}
	} else {
		for queue, _ := range queues {
			fmt.Println(queue)
		}
	}
}
