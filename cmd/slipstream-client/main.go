package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	slipstream "github.com/TheCitadelX/libslipstream-go"
)

func main() {
	var (
		resolversCSV      = flag.String("resolvers", "", "comma-separated DNS resolver addresses")
		resolver          = flag.String("resolver", "", "single DNS resolver address")
		domain            = flag.String("domain", "", "tunnel domain")
		socksListen       = flag.String("socks", "127.0.0.1:1080", "local SOCKS5 listen address")
		fingerprint       = flag.String("cert-fingerprint", "", "server certificate SHA-256 fingerprint")
		pinnedCertPath    = flag.String("pinned-cert", "", "server certificate PEM pin path")
		serverName        = flag.String("server-name", "", "TLS server name for platform verification")
		allowInsecure     = flag.Bool("allow-insecure", false, "disable TLS verification; local testing only")
		initialPacketSize = flag.Uint("initial-packet-size", 1200, "QUIC initial packet size")
	)
	flag.Parse()

	var pinnedCertPEM []byte
	if strings.TrimSpace(*pinnedCertPath) != "" {
		var err error
		pinnedCertPEM, err = os.ReadFile(*pinnedCertPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	if *initialPacketSize > 65535 {
		log.Fatal("initial-packet-size must be <= 65535")
	}

	client, err := slipstream.NewClient(slipstream.ClientConfig{
		ResolverAddress:   strings.TrimSpace(*resolver),
		Resolvers:         splitCSV(*resolversCSV),
		Domain:            strings.TrimSpace(*domain),
		TCPListenAddress:  strings.TrimSpace(*socksListen),
		CertFingerprint:   strings.TrimSpace(*fingerprint),
		PinnedCertPEM:     pinnedCertPEM,
		ServerName:        strings.TrimSpace(*serverName),
		AllowInsecure:     *allowInsecure,
		InitialPacketSize: uint16(*initialPacketSize),
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Start(); err != nil {
		log.Fatal(err)
	}
	defer client.Stop()

	proxyAddr, err := client.StartSOCKS5(*socksListen)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("slipstream client connected to %s via %s", strings.TrimSpace(*domain), resolverList(*resolver, *resolversCSV))
	log.Printf("SOCKS5 proxy listening on %s", proxyAddr)

	waitSignal()
	if err := client.StopSOCKS5(); err != nil {
		log.Fatal(err)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func resolverList(resolver, resolversCSV string) string {
	resolvers := splitCSV(resolversCSV)
	if strings.TrimSpace(resolver) != "" {
		resolvers = append([]string{strings.TrimSpace(resolver)}, resolvers...)
	}
	if len(resolvers) == 0 {
		return "<none>"
	}
	return strings.Join(resolvers, ",")
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	fmt.Println()
}
