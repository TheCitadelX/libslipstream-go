package dns

func BuildQName(payload []byte, domain string) (string, error) {
	domain = trimTrailingDot(domain)
	if domain == "" {
		return "", errf("domain must not be empty")
	}
	maxPayload, err := MaxPayloadLenForDomain(domain)
	if err != nil {
		return "", err
	}
	if len(payload) > maxPayload {
		return "", errf("payload too large for domain")
	}
	base32 := EncodeBase32(payload)
	dotted := Dotify(base32)
	return dotted + "." + domain + ".", nil
}

func MaxPayloadLenForDomain(domain string) (int, error) {
	domain = trimTrailingDot(domain)
	if domain == "" {
		return 0, errf("domain must not be empty")
	}
	if len(domain) > maxDNSNameLen {
		return 0, errf("domain too long")
	}

	maxDottedLen := maxDNSNameLen - len(domain) - 1
	if maxDottedLen <= 0 {
		return 0, nil
	}

	maxBase32Len := 0
	for l := 1; l <= maxDottedLen; l++ {
		dots := (l - 1) / 57
		if l+dots > maxDottedLen {
			break
		}
		maxBase32Len = l
	}

	maxPayload := (maxBase32Len * 5) / 8
	for maxPayload > 0 && base32Len(maxPayload) > maxBase32Len {
		maxPayload--
	}
	return maxPayload, nil
}

func base32Len(payloadLen int) int {
	if payloadLen == 0 {
		return 0
	}
	return (payloadLen*8 + 4) / 5
}

func ComputeMTU(domain string) (uint16, error) {
	domain = trimTrailingDot(domain)
	if len(domain) >= 240 {
		return 0, errf("domain name is too long for DNS transport")
	}
	mtu := uint16((240.0 - float64(len(domain))) / 1.6)
	if mtu == 0 {
		return 0, errf("mtu computed to zero; check domain length")
	}
	return mtu, nil
}
