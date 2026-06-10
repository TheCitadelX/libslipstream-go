package dns

import (
	"strings"
)

const maxDNSNameLen = 253

func extractSubdomain(qname, domain string) (string, RCode) {
	domain = strings.TrimRight(domain, ".")
	if domain == "" {
		return "", RCodeNameError
	}

	suffix := "." + domain + "."
	if !strings.HasSuffix(strings.ToLower(qname), strings.ToLower(suffix)) {
		return "", RCodeNameError
	}
	if len(qname) <= len(domain)+2 {
		return "", RCodeNameError
	}

	dataLen := len(qname) - len(domain) - 2
	subdomain := qname[:dataLen]
	if subdomain == "" {
		return "", RCodeNameError
	}
	return subdomain, RCodeOK
}

func extractSubdomainMulti(qname string, domains []string) (string, RCode) {
	qnameTrimmed := strings.TrimRight(qname, ".")
	if qnameTrimmed == "" {
		return "", RCodeNameError
	}
	qnameLower := strings.ToLower(qnameTrimmed)

	bestDomain := ""
	bestLen := 0
	bestEmpty := false
	for _, domain := range domains {
		domainTrimmed := strings.TrimRight(domain, ".")
		if domainTrimmed == "" {
			continue
		}
		domainLower := strings.ToLower(domainTrimmed)
		isExact := qnameLower == domainLower
		isSuffix := !isExact &&
			len(qnameLower) > len(domainLower) &&
			strings.HasSuffix(qnameLower, domainLower) &&
			qnameLower[len(qnameLower)-len(domainLower)-1] == '.'
		if !isExact && !isSuffix {
			continue
		}
		if len(domainTrimmed) > bestLen {
			bestLen = len(domainTrimmed)
			bestDomain = domainTrimmed
			bestEmpty = isExact
		}
	}
	if bestDomain == "" || bestEmpty {
		return "", RCodeNameError
	}
	return extractSubdomain(qname, bestDomain)
}

func parseName(packet []byte, start int) (string, int, error) {
	labels := make([]string, 0, 4)
	offset := start
	jumped := false
	endOffset := start
	seen := make(map[int]struct{})
	depth := 0
	nameLen := 0

	for {
		if offset >= len(packet) {
			return "", 0, errf("name out of range")
		}
		l := packet[offset]
		if l&0xC0 == 0xC0 {
			if offset+1 >= len(packet) {
				return "", 0, errf("truncated pointer")
			}
			ptr := int(l&0x3F)<<8 | int(packet[offset+1])
			if ptr >= len(packet) {
				return "", 0, errf("pointer out of range")
			}
			if _, ok := seen[ptr]; ok {
				return "", 0, errf("pointer loop")
			}
			seen[ptr] = struct{}{}
			if !jumped {
				endOffset = offset + 2
				jumped = true
			}
			offset = ptr
			depth++
			if depth > 16 {
				return "", 0, errf("pointer depth exceeded")
			}
			continue
		}
		if l == 0 {
			offset++
			if !jumped {
				endOffset = offset
			}
			break
		}
		if l > 63 {
			return "", 0, errf("label too long")
		}
		offset++
		end := offset + int(l)
		if end > len(packet) {
			return "", 0, errf("label out of range")
		}
		if len(labels) > 0 {
			nameLen++
		}
		nameLen += int(l)
		if nameLen > maxDNSNameLen {
			return "", 0, errf("name too long")
		}
		labels = append(labels, string(packet[offset:end]))
		offset = end
		if !jumped {
			endOffset = offset
		}
	}

	if len(labels) == 0 {
		return ".", endOffset, nil
	}
	return strings.Join(labels, ".") + ".", endOffset, nil
}

func encodeName(name string, out *[]byte) error {
	if name == "." {
		*out = append(*out, 0)
		return nil
	}

	trimmed := strings.TrimRight(name, ".")
	nameLen := 0
	first := true
	for _, label := range strings.Split(trimmed, ".") {
		if label == "" {
			return errf("empty label")
		}
		if len(label) > 63 {
			return errf("label too long")
		}
		if !first {
			nameLen++
		}
		nameLen += len(label)
		if nameLen > maxDNSNameLen {
			return errf("name too long")
		}
		*out = append(*out, byte(len(label)))
		*out = append(*out, label...)
		first = false
	}
	*out = append(*out, 0)
	return nil
}
