package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/tsingson/fasthttploader/report"
	"github.com/valyala/fasthttp"
)

var (
	method      = flag.String("m", "GET", "Set HTTP method")
	headers     = flag.String("h", "", "Set headers")
	body        = flag.String("b", "", "Set body")
	accept      = flag.String("A", "", "Set Accept headers")
	contentType = flag.String("T", "text/html", "Set content-type headers")

	fileName = flag.String("r", "report.html", "Set filename to store final report")
	web      = flag.Bool("web", false, "Auto open generated report at browser")

	d = flag.Duration("d", 30*time.Second, "Cant be less than 20sec")
	t = flag.Duration("t", 5*time.Second, "Request timeout")
	q = flag.Int("q", 0, "Request per second limit. Detect automatically, if not setted")
	c = flag.Int("c", 500, "Number of supposed clients")

	debug              = flag.Bool("debug", false, "Print debug messages if true")
	disableKeepAlive   = flag.Bool("k", false, "Disable keepalive if true")
	disableCompression = flag.Bool("disable-compression", false, "Disables compression if true")
	successStatusCode  = flag.Int("successStatusCode", fasthttp.StatusOK, "Status code on which a successful request would be determined")

	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to this file")
)

var usage = `Usage: fasthttploader [options...] <url>
Notice: fasthttploader would force aggressive burst stages before testing to detect max qps and number for clients.
To avoid this you need to set -c and -q parameters.
Options:
`

var req = new(fasthttp.Request)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit("")
	}

	if *d < time.Second*20 {
		usageAndExit("Duration cant be less than 20s")
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	applyHeaders()
	req.AppendBodyString(*body)
	run()

	if *web {
		err := report.OpenBrowser(*fileName)
		if err != nil {
			fmt.Printf("Can't open browser to display report: %s", err)
		}
	} else {
		command, err := report.PrintOpenBrowser(*fileName)
		if err != nil {
			fmt.Printf("Can't generate command to display report in browser: %s", err)
		}
		fmt.Printf("Check test results by executing next command:\n %s\n", command)
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}

var re = regexp.MustCompile("^([\\w-]+):\\s*(.+)")

func applyHeaders() {
	var url string
	req.Header.SetContentType(*contentType)
	if *headers != "" {
		headers := strings.Split(*headers, ";")
		for _, h := range headers {
			matches := re.FindStringSubmatch(h)
			if len(matches) < 1 {
				usageAndExit(fmt.Sprintf("could not parse the provided input; input = %v", h))
			}
			req.Header.Set(matches[1], matches[2])
		}
	}
	if *accept != "" {
		req.Header.Set("Accept", *accept)
	}
	url = flag.Args()[0]
	req.Header.SetMethod(strings.ToUpper(*method))
	req.Header.SetRequestURI(url)
	if !*disableCompression {
		req.Header.Set("Accept-Encoding", "gzip")
	}
	if *disableKeepAlive {
		req.Header.Set("Connection", "close")
	} else {
		req.Header.Set("Connection", "keep-alive")
	}
}

func usageAndExit(msg string) {
	flag.Usage()
	if msg != "" {
		fmt.Print("----------------------------\nErr: ")
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	os.Exit(1)
}
