package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Scanner struct {
	OutputFile      string
	Target          string
	TargetFile      string
	DefaultScheme   string
	UserAgent       string
	BadHostName     string
	Delay           time.Duration
	Timeout         time.Duration
	Threads         int
	MinScore        float64
	FollowRedirects bool

	targets []string
	results []string
}

var buildVersion = "1.0.0"

func (s *Scanner) Run() error {
	if s.TargetFile != "" {
		targets, err := readFileByLine(s.TargetFile)
		if err != nil {
			return err
		}
		s.targets = targets
	}
	if s.Target != "" {
		s.targets = append(s.targets, strings.Split(s.Target, ",")...)
	}

	if len(s.targets) == 0 {
		return nil
	}

	jobs := make(chan *url.URL)

	wg := &sync.WaitGroup{}
	wg.Add(s.Threads)
	for i := 1; i <= s.Threads; i++ {
		go func(i int) {
			defer wg.Done()

			for j := range jobs {
				s.doWork(j)
			}
		}(i)
	}

	nt := len(s.targets)
	for i, _target := range s.targets {
		if target, err := s.prepareJob(_target); err == nil {
			jobs <- target
			if nt > i+1 {
				time.Sleep(s.Delay)
			}
		} else {
			color.Red("[error] %s", err.Error())
		}
	}
	close(jobs)

	wg.Wait()

	return nil
}

func (s *Scanner) prepareJob(target string) (*url.URL, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	if u != nil {
		if u.Scheme == "" {
			target = s.DefaultScheme + "://" + target
			u, err = url.Parse(target)
		}
	}
	return u, err
}

func (s *Scanner) doWork(target *url.URL) {
	if score, status := s.work(target); score >= s.MinScore {
		if strings.Contains(status, "200") {
			color.Green("[success] %s [injection]", status)
		} else {
			color.Cyan("[info] %s [injection]", status)
		}
	} else if strings.Contains(status, "200") {
		color.Yellow("[info] %s [none]", status)
	} else if status != "" {
		color.Magenta("[failed] %s []", status)
	}
}

func (s *Scanner) work(target *url.URL) (float64, string) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: s.Timeout,
	}

	if s.FollowRedirects == false {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	ip := resolveHost(target.Host)
	if ip == nil {
		color.Red("[error] host not unreachable %s", target.String())
		return 0, ""
	}

	_target, _ := url.Parse(strings.Replace(target.String(), target.Host, ip.String(), 1))
	req, _ := http.NewRequest(http.MethodGet, _target.String(), nil)
	req.Host = target.Host
	req.Header.Set("user-agent", s.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		color.Red("[error] %s", err.Error())
		return 0, ""
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	line := fmt.Sprintf("%s [%d] [%d]", target.String(), resp.StatusCode, len(bodyBytes))

	req.Host = s.BadHostName
	resp, err = client.Do(req)
	if err != nil {
		color.Red("[error] %s", err.Error())
		return 0, ""
	}
	defer resp.Body.Close()

	bodyBytes2, _ := io.ReadAll(resp.Body)
	delta := compare(string(bodyBytes), string(bodyBytes2))
	line = fmt.Sprintf("%s [%d] [%d] [%.2f]", line, resp.StatusCode, len(bodyBytes2), delta)

	if location, err := resp.Location(); err == nil && location != nil {
		if location.Host == s.BadHostName {
			color.Green("[success] %s [redirect]", line)
		}
	}

	return delta, line
}

func compare(str1, str2 string) float64 {
	parts1 := strings.Split(str1, "\n")
	parts2 := strings.Split(str2, "\n")
	var check bool
	result := 0
	for _, s := range parts1 {
		parts2, check = _compare(parts2, s)
		if check {
			result++
		}
	}
	return float64(result) / float64(len(parts1)) * 100.0
}

func _compare(parts []string, str string) ([]string, bool) {
	for i, _s := range parts {
		if str == _s {
			if len(parts) > i {
				return append(parts[:i], parts[i+1:]...), true
			}
			return parts[:i], true
		}
	}
	return parts, false
}

//
// resolveHost
// @Description: Resolve a target and get the first associated ip address
// @param domain string
// @return net.IP
func resolveHost(domain string) net.IP {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil
	}
	for _, ip := range ips {
		return ip
	}
	return nil
}

//
// readFileByLine
// @Description: Reads a file line by line and returns them as a string list
// @param filename string
// @return []string
// @return error
func readFileByLine(filename string) ([]string, error) {
	lines := make([]string, 0)
	f, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return lines, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err = sc.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	s := &Scanner{
		OutputFile:      "",
		BadHostName:     "somethingbadthatdoesntexist-hopefully.com",
		DefaultScheme:   "https",
		UserAgent:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36",
		Delay:           0,
		Timeout:         10 * time.Second,
		Threads:         10,
		MinScore:        90.0,
		FollowRedirects: false,
		targets:         make([]string, 0),
		results:         make([]string, 0),
	}

	flag.CommandLine.StringVar(&s.OutputFile, "output", s.OutputFile, "File to store all outputs")
	flag.CommandLine.StringVar(&s.Target, "target", s.Target, "Targets to scan")
	flag.CommandLine.StringVar(&s.UserAgent, "user-agent", s.UserAgent, "Set a custom user agent")
	flag.CommandLine.BoolVar(&s.FollowRedirects, "redirects", s.FollowRedirects, "Follow all redirects")
	flag.CommandLine.StringVar(&s.TargetFile, "target-file", s.TargetFile, "File containing a list of targets")
	flag.CommandLine.StringVar(&s.DefaultScheme, "scheme", s.DefaultScheme, "Default url scheme")
	flag.CommandLine.DurationVar(&s.Delay, "delay", s.Delay, "Delay between requests")
	flag.CommandLine.DurationVar(&s.Timeout, "timeout", s.Timeout, "Request timeout")
	flag.CommandLine.IntVar(&s.Threads, "threads", s.Threads, "Number of threads")
	flag.CommandLine.Float64Var(&s.MinScore, "score", s.MinScore, "Percentage of response lines that have to be identical")
	flag.CommandLine.StringVar(&s.BadHostName, "host-name", s.BadHostName, "Fake hostname used to verify host header injection")

	sv := flag.Bool("version", false, "Show version and exit")
	nc := flag.Bool("no-color", false, "Disable color output")
	flag.Parse()

	if *nc {
		color.NoColor = true // disables colorized output
	}

	if *sv {
		fmt.Printf("version: %s\n", color.CyanString(buildVersion))
		os.Exit(0)
	}

	if err := s.Run(); err != nil {
		fmt.Printf("[error] %s\n", err.Error())
		os.Exit(1)
	}
}
