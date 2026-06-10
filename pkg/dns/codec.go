package dns

func DecodeQuery(packet []byte, domain string) (DecodedQuery, error) {
	return DecodeQueryWithDomains(packet, []string{domain})
}

func DecodeQueryWithDomains(packet []byte, domains []string) (DecodedQuery, error) {
	h, ok := parseHeader(packet)
	if !ok {
		return DecodedQuery{}, DropError{}
	}

	if h.isResponse {
		question, err := parseQuestionForReply(packet, h.qdcount, h.offset)
		if err != nil {
			return DecodedQuery{}, err
		}
		return DecodedQuery{}, &QueryReplyError{
			ID:       h.id,
			RD:       h.rd,
			CD:       h.cd,
			Question: question,
			RCode:    RCodeFormatError,
		}
	}

	if h.qdcount != 1 {
		question, err := parseQuestionForReply(packet, h.qdcount, h.offset)
		if err != nil {
			return DecodedQuery{}, err
		}
		return DecodedQuery{}, &QueryReplyError{
			ID:       h.id,
			RD:       h.rd,
			CD:       h.cd,
			Question: question,
			RCode:    RCodeFormatError,
		}
	}

	question, _, err := parseQuestion(packet, h.offset)
	if err != nil {
		return DecodedQuery{}, DropError{}
	}

	if question.Type != RRTXT {
		return DecodedQuery{}, &QueryReplyError{
			ID:       h.id,
			RD:       h.rd,
			CD:       h.cd,
			Question: &question,
			RCode:    RCodeNameError,
		}
	}

	subdomainRaw, rcode := extractSubdomainMulti(question.Name, domains)
	if rcode != RCodeOK {
		return DecodedQuery{}, &QueryReplyError{
			ID:       h.id,
			RD:       h.rd,
			CD:       h.cd,
			Question: &question,
			RCode:    rcode,
		}
	}

	undotted := Undotify(subdomainRaw)
	if undotted == "" {
		return DecodedQuery{}, &QueryReplyError{
			ID:       h.id,
			RD:       h.rd,
			CD:       h.cd,
			Question: &question,
			RCode:    RCodeNameError,
		}
	}

	payload, err := DecodeBase32(undotted)
	if err != nil {
		return DecodedQuery{}, &QueryReplyError{
			ID:       h.id,
			RD:       h.rd,
			CD:       h.cd,
			Question: &question,
			RCode:    RCodeServerFailure,
		}
	}

	return DecodedQuery{
		ID:       h.id,
		RD:       h.rd,
		CD:       h.cd,
		Question: question,
		Payload:  payload,
	}, nil
}

func EncodeQuery(params QueryParams) ([]byte, error) {
	out := make([]byte, 0, 256)
	var flags uint16
	if !params.IsQuery {
		flags |= 0x8000
	}
	if params.RD {
		flags |= 0x0100
	}
	if params.CD {
		flags |= 0x0010
	}

	writeU16(&out, params.ID)
	writeU16(&out, flags)
	writeU16(&out, params.QDCount)
	writeU16(&out, 0)
	writeU16(&out, 0)
	writeU16(&out, 1)

	if params.QDCount > 0 {
		if err := encodeName(params.QName, &out); err != nil {
			return nil, err
		}
		writeU16(&out, params.QType)
		writeU16(&out, params.QClass)
	}

	encodeOPTRecord(&out)
	return out, nil
}

func EncodePayloadQuery(id uint16, payload []byte, domain string) ([]byte, string, error) {
	qname, err := BuildQName(payload, domain)
	if err != nil {
		return nil, "", err
	}
	packet, err := EncodeQuery(QueryParams{
		ID:      id,
		QName:   qname,
		QType:   RRTXT,
		QClass:  ClassIN,
		RD:      true,
		QDCount: 1,
		IsQuery: true,
	})
	return packet, qname, err
}

func EncodeResponse(params ResponseParams) ([]byte, error) {
	payloadLen := len(params.Payload)
	rcode := RCodeNameError
	if payloadLen > 0 {
		rcode = RCodeOK
	}
	if params.RCode != nil {
		rcode = *params.RCode
	}

	var ancount uint16
	if payloadLen > 0 && rcode == RCodeOK {
		ancount = 1
	}

	out := make([]byte, 0, 256)
	flags := uint16(0x8000 | 0x0400 | uint16(rcode))
	if params.RD {
		flags |= 0x0100
	}
	if params.CD {
		flags |= 0x0010
	}

	writeU16(&out, params.ID)
	writeU16(&out, flags)
	writeU16(&out, 1)
	writeU16(&out, ancount)
	writeU16(&out, 0)
	writeU16(&out, 1)

	if err := encodeName(params.Question.Name, &out); err != nil {
		return nil, err
	}
	writeU16(&out, params.Question.Type)
	writeU16(&out, params.Question.Class)

	if ancount == 1 {
		out = append(out, 0xC0, 0x0C)
		writeU16(&out, params.Question.Type)
		writeU16(&out, params.Question.Class)
		writeU32(&out, 60)
		chunkCount := (payloadLen + 254) / 255
		rdataLen := payloadLen + chunkCount
		if rdataLen > 0xffff {
			return nil, errf("payload too long")
		}
		writeU16(&out, uint16(rdataLen))
		remaining := payloadLen
		cursor := 0
		for remaining > 0 {
			chunkLen := min(remaining, 255)
			out = append(out, byte(chunkLen))
			out = append(out, params.Payload[cursor:cursor+chunkLen]...)
			cursor += chunkLen
			remaining -= chunkLen
		}
	}

	encodeOPTRecord(&out)
	return out, nil
}

func DecodeResponse(packet []byte) []byte {
	h, ok := parseHeader(packet)
	if !ok || !h.isResponse || !h.rcodeOK || h.rcode != RCodeOK || h.ancount != 1 {
		return nil
	}

	offset := h.offset
	for i := uint16(0); i < h.qdcount; i++ {
		var err error
		_, offset, err = parseName(packet, offset)
		if err != nil || offset+4 > len(packet) {
			return nil
		}
		offset += 4
	}

	var err error
	_, offset, err = parseName(packet, offset)
	if err != nil || offset+10 > len(packet) {
		return nil
	}
	qtype := readU16(packet, offset)
	offset += 2
	offset += 2
	offset += 4
	rdlen := int(readU16(packet, offset))
	offset += 2
	if offset+rdlen > len(packet) || rdlen < 1 || qtype != RRTXT {
		return nil
	}

	remaining := rdlen
	cursor := offset
	out := make([]byte, 0, rdlen)
	for remaining > 0 {
		txtLen := int(packet[cursor])
		cursor++
		remaining--
		if txtLen > remaining {
			return nil
		}
		out = append(out, packet[cursor:cursor+txtLen]...)
		cursor += txtLen
		remaining -= txtLen
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func IsResponse(packet []byte) bool {
	h, ok := parseHeader(packet)
	return ok && h.isResponse
}

func ReplyFromDecodeError(err error) (*ResponseParams, bool) {
	reply, ok := err.(*QueryReplyError)
	if !ok || reply.Question == nil {
		return nil, false
	}
	return &ResponseParams{
		ID:       reply.ID,
		RD:       reply.RD,
		CD:       reply.CD,
		Question: *reply.Question,
		RCode:    &reply.RCode,
	}, true
}

func encodeOPTRecord(out *[]byte) {
	*out = append(*out, 0)
	writeU16(out, RROPT)
	writeU16(out, EDNSUDPPayload)
	writeU32(out, 0)
	writeU16(out, 0)
}

func trimTrailingDot(domain string) string {
	for len(domain) > 0 && domain[len(domain)-1] == '.' {
		domain = domain[:len(domain)-1]
	}
	return domain
}
