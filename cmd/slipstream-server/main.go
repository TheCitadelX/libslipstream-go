package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	slipstream "github.com/TheCitadelX/libslipstream-go"
)

func main() {
	var (
		dnsListen    = flag.String("dns", ":53", "UDP DNS listen address")
		domain       = flag.String("domain", "", "primary tunnel domain")
		domainsCSV   = flag.String("domains", "", "comma-separated tunnel domains")
		target       = flag.String("target", "", "optional fallback TCP target address")
		certPath     = flag.String("cert", "", "server certificate PEM path")
		keyPath      = flag.String("key", "", "server private key PEM path")
		responseWait = flag.Duration("response-wait", 50*time.Millisecond, "DNS response wait timeout")
		queueSize    = flag.Int("queue", 8192, "packet queue size")
	)
	flag.Parse()

	certPEM, err := readRequiredFile(*certPath, "cert")
	if err != nil {
		log.Fatal(err)
	}
	keyPEM, err := readRequiredFile(*keyPath, "key")
	if err != nil {
		log.Fatal(err)
	}

	server, err := slipstream.NewServer(slipstream.ServerConfig{
		DNSListenAddress: strings.TrimSpace(*dnsListen),
		TargetAddress:    strings.TrimSpace(*target),
		Domain:           strings.TrimSpace(*domain),
		Domains:          splitCSV(*domainsCSV),
		CertPEM:          certPEM,
		KeyPEM:           keyPEM,
		ResponseWait:     *responseWait,
		PacketQueueSize:  *queueSize,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
	log.Printf("slipstream server listening on %s for domains %s", server.LocalDNSAddress(), domainList(*domain, *domainsCSV))

	waitSignal()
	if err := server.Stop(); err != nil {
		log.Fatal(err)
	}
}

func readRequiredFile(path, name string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%s path is required", name)
	}
	return os.ReadFile(path)
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

func domainList(domain, domainsCSV string) string {
	domains := splitCSV(domainsCSV)
	if strings.TrimSpace(domain) != "" {
		domains = append([]string{strings.TrimSpace(domain)}, domains...)
	}
	return strings.Join(domains, ",")
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
