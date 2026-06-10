package dns

func Dotify(input string) string {
	if input == "" {
		return ""
	}
	bytes := []byte(input)
	length := len(bytes)
	dots := (length - 1) / 57
	newLength := length + dots

	buf := make([]byte, newLength)
	copy(buf, bytes)

	src := length - 1
	dst := newLength - 1
	nextDot := length - (length % 57)
	if length%57 == 0 {
		nextDot = length - 57
	}
	currentPos := length

	for currentPos > 0 {
		if currentPos == nextDot {
			buf[dst] = '.'
			dst--
			if nextDot >= 57 {
				nextDot -= 57
			} else {
				nextDot = 0
			}
			currentPos--
			continue
		}

		buf[dst] = buf[src]
		dst--
		src--
		currentPos--
	}

	return string(buf)
}

func Undotify(input string) string {
	out := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] != '.' {
			out = append(out, input[i])
		}
	}
	return string(out)
}
