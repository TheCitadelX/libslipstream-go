package dns

import "encoding/binary"

type header struct {
	id         uint16
	isResponse bool
	rd         bool
	cd         bool
	qdcount    uint16
	ancount    uint16
	rcode      RCode
	rcodeOK    bool
	offset     int
}

func parseHeader(packet []byte) (header, bool) {
	if len(packet) < 12 {
		return header{}, false
	}
	flags := readU16(packet, 2)
	rcode, ok := rcodeFromByte(byte(flags & 0x000f))
	return header{
		id:         readU16(packet, 0),
		isResponse: flags&0x8000 != 0,
		rd:         flags&0x0100 != 0,
		cd:         flags&0x0010 != 0,
		qdcount:    readU16(packet, 4),
		ancount:    readU16(packet, 6),
		rcode:      rcode,
		rcodeOK:    ok,
		offset:     12,
	}, true
}

func parseQuestion(packet []byte, offset int) (Question, int, error) {
	name, offset, err := parseName(packet, offset)
	if err != nil {
		return Question{}, 0, errf("bad name")
	}
	if offset+4 > len(packet) {
		return Question{}, 0, errf("truncated question")
	}
	question := Question{
		Name:  name,
		Type:  readU16(packet, offset),
		Class: readU16(packet, offset+2),
	}
	return question, offset + 4, nil
}

func parseQuestionForReply(packet []byte, qdcount uint16, offset int) (*Question, error) {
	if qdcount == 0 {
		return nil, nil
	}
	question, _, err := parseQuestion(packet, offset)
	if err != nil {
		return nil, DropError{}
	}
	return &question, nil
}

func readU16(packet []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(packet[offset : offset+2])
}

func readU32(packet []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(packet[offset : offset+4])
}

func writeU16(out *[]byte, value uint16) {
	*out = binary.BigEndian.AppendUint16(*out, value)
}

func writeU32(out *[]byte, value uint32) {
	*out = binary.BigEndian.AppendUint32(*out, value)
}
