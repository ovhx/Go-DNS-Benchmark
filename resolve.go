package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	// Prompt for the input and output filenames
	var inputFilename, outputFilename string
	fmt.Print("Enter Domain list to resolve: ")
	fmt.Scanln(&inputFilename)
	fmt.Print("Enter IP list filename (Output): ")
	fmt.Scanln(&outputFilename)

	// Open the input file
	input, err := os.Open(inputFilename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer input.Close()

	// Create the output file
	output, err := os.Create(outputFilename)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer output.Close()

	// Set the DNS server to use
	var dnsServer string
	fmt.Print("Enter DNS server to use (Example: 1.1.1.1): ")
	fmt.Scanln(&dnsServer)
	if dnsServer == "" {
		dnsServer = "1.1.1.1"
	}
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Second * 10,
		}
		return d.DialContext(ctx, "udp", dnsServer)
	}

	// Read the input file line by line
	scanner := bufio.NewScanner(input)
	var wg sync.WaitGroup
	domainsCh := make(chan string)
	numThreads := 0
	fmt.Print("Enter number of threads to use: ")
	fmt.Scanln(&numThreads)
	if numThreads < 1 {
		numThreads = 1
	} else if numThreads > 100 {
		numThreads = 100
	}
	var sumTime time.Duration
	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func() {
			for domain := range domainsCh {
				// Look up the IP address for the domain
				start := time.Now()
				ip, err := net.DefaultResolver.LookupIP(context.Background(), "ip", domain)
				if err != nil {
					fmt.Println(err)
					continue
				}

				// Write the IP address to the output file
				output.WriteString(ip[0].String() + "\n")

				// Print the time it took to resolve the domain
				elapsed := time.Since(start)
				fmt.Printf("%s: %s\n", domain, elapsed)
				sumTime += elapsed
			}
			wg.Done()
		}()
	}

	// Enqueue domains to be resolved
	numResolved := 0
	for scanner.Scan() {
		domain := scanner.Text()
		domainsCh <- domain
		numResolved++
	}
	close(domainsCh)
	wg.Wait()
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

	// Print statistics
	fmt.Printf("\nTotal domains resolved: %d\n", numResolved)
	fmt.Printf("Domains resolved successfully: %d\n", numResolved-len(domainsCh))
	fmt.Printf("Domains that failed to resolve or timed out: %d\n", len(domainsCh))
	if numResolved > 0 {
		fmt.Printf("Average resolve time: %s\n", time.Duration(int64(sumTime)/int64(numResolved)))
	}
}
