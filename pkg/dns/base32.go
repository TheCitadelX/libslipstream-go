package dns

import "errors"

const encodeTable = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

var (
	ErrInvalidBase32Length  = errors.New("invalid base32 length")
	ErrInvalidBase32Char    = errors.New("invalid base32 character")
	ErrInvalidBase32Padding = errors.New("invalid base32 padding")
)

func EncodeBase32(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	out := make([]byte, 0, (len(input)*8+4)/5)
	var buffer uint32
	var bits uint8
	for _, b := range input {
		buffer = (buffer << 8) | uint32(b)
		bits += 8
		for bits >= 5 {
			shift := bits - 5
			out = append(out, encodeTable[(buffer>>shift)&0x1f])
			bits -= 5
		}
	}
	if bits > 0 {
		out = append(out, encodeTable[(buffer<<(5-bits))&0x1f])
	}
	return string(out)
}

func DecodeBase32(input string) ([]byte, error) {
	if input == "" {
		return []byte{}, nil
	}

	cleaned := make([]byte, 0, len(input))
	sawPad := false
	for i := 0; i < len(input); i++ {
		b := input[i]
		if b == '.' {
			continue
		}
		if b == '=' {
			sawPad = true
			cleaned = append(cleaned, b)
			continue
		}
		if sawPad {
			return nil, ErrInvalidBase32Padding
		}
		cleaned = append(cleaned, b)
	}
	if len(cleaned) == 0 {
		return []byte{}, nil
	}

	dataLen := len(cleaned)
	pad := 0
	for dataLen > 0 && cleaned[dataLen-1] == '=' {
		pad++
		dataLen--
	}
	if pad > 0 {
		for _, b := range cleaned[:dataLen] {
			if b == '=' {
				return nil, ErrInvalidBase32Padding
			}
		}
		if len(cleaned) < 8 || len(cleaned)%8 != 0 || pad > 6 {
			return nil, ErrInvalidBase32Padding
		}
	}

	data := cleaned[:dataLen]
	rem := len(data) % 8
	if rem != 0 && rem != 2 && rem != 4 && rem != 5 && rem != 7 {
		return nil, ErrInvalidBase32Length
	}

	out := make([]byte, 0, len(data)*5/8+4)
	index := 0
	for index+8 <= len(data) {
		v1, err := decodeBase32Value(data[index])
		if err != nil {
			return nil, err
		}
		v2, err := decodeBase32Value(data[index+1])
		if err != nil {
			return nil, err
		}
		v3, err := decodeBase32Value(data[index+2])
		if err != nil {
			return nil, err
		}
		v4, err := decodeBase32Value(data[index+3])
		if err != nil {
			return nil, err
		}
		v5, err := decodeBase32Value(data[index+4])
		if err != nil {
			return nil, err
		}
		v6, err := decodeBase32Value(data[index+5])
		if err != nil {
			return nil, err
		}
		v7, err := decodeBase32Value(data[index+6])
		if err != nil {
			return nil, err
		}
		v8, err := decodeBase32Value(data[index+7])
		if err != nil {
			return nil, err
		}

		out = append(out,
			(v1<<3)|(v2>>2),
			(v2<<6)|(v3<<1)|(v4>>4),
			(v4<<4)|(v5>>1),
			(v5<<7)|(v6<<2)|(v7>>3),
			(v7<<5)|v8,
		)
		index += 8
	}

	remaining := len(data) - index
	if remaining > 0 {
		v1, err := decodeBase32Value(data[index])
		if err != nil {
			return nil, err
		}
		v2, err := decodeBase32Value(data[index+1])
		if err != nil {
			return nil, err
		}
		out = append(out, (v1<<3)|(v2>>2))
		if remaining == 2 {
			return out, nil
		}

		v3, err := decodeBase32Value(data[index+2])
		if err != nil {
			return nil, err
		}
		v4, err := decodeBase32Value(data[index+3])
		if err != nil {
			return nil, err
		}
		out = append(out, (v2<<6)|(v3<<1)|(v4>>4))
		if remaining == 4 {
			return out, nil
		}

		v5, err := decodeBase32Value(data[index+4])
		if err != nil {
			return nil, err
		}
		out = append(out, (v4<<4)|(v5>>1))
		if remaining == 5 {
			return out, nil
		}

		v6, err := decodeBase32Value(data[index+5])
		if err != nil {
			return nil, err
		}
		v7, err := decodeBase32Value(data[index+6])
		if err != nil {
			return nil, err
		}
		out = append(out, (v5<<7)|(v6<<2)|(v7>>3))
	}

	return out, nil
}

func decodeBase32Value(b byte) (byte, error) {
	switch {
	case b >= 'A' && b <= 'Z':
		return b - 'A', nil
	case b >= 'a' && b <= 'z':
		return b - 'a', nil
	case b >= '2' && b <= '7':
		return b - '2' + 26, nil
	default:
		return 0, ErrInvalidBase32Char
	}
}
