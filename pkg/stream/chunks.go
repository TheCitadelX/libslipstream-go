package stream

import "sort"

type Chunk struct {
	Offset uint64
	Data   []byte
}

type RecvState struct {
	ConsumedOffset uint64
	SentOffset     uint64
	BufferedBytes  int
	PendingFIN     *uint64
	FINEnqueued    bool
	Chunks         map[uint64][]byte
}

func NewRecvState() *RecvState {
	return &RecvState{Chunks: make(map[uint64][]byte)}
}

func InsertChunk(chunks map[uint64][]byte, sentOffset, offset uint64, data []byte) int {
	if len(data) == 0 {
		return 0
	}
	if chunks == nil {
		return 0
	}

	start := offset
	bytes := data
	if start < sentOffset {
		delta := int(sentOffset - start)
		if delta >= len(bytes) {
			return 0
		}
		bytes = bytes[delta:]
		start = sentOffset
	}

	end := start + uint64(len(bytes))
	if end == start {
		return 0
	}

	keys := make([]uint64, 0, len(chunks))
	for k := range chunks {
		if k < end {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	cursor := start
	inserts := make([]Chunk, 0, 2)
	inserted := 0
	for _, segStart := range keys {
		segData := chunks[segStart]
		segEnd := segStart + uint64(len(segData))
		if segEnd <= cursor {
			continue
		}
		if segStart > cursor {
			gapEnd := min64(segStart, end)
			gapLen := int(gapEnd - cursor)
			gapOffset := int(cursor - start)
			inserts = append(inserts, Chunk{
				Offset: cursor,
				Data:   append([]byte(nil), bytes[gapOffset:gapOffset+gapLen]...),
			})
			inserted += gapLen
			cursor = gapEnd
		}
		if segEnd > cursor {
			cursor = segEnd
		}
		if cursor >= end {
			break
		}
	}

	if cursor < end {
		gapOffset := int(cursor - start)
		inserts = append(inserts, Chunk{
			Offset: cursor,
			Data:   append([]byte(nil), bytes[gapOffset:]...),
		})
		inserted += len(bytes) - gapOffset
	}

	for _, insert := range inserts {
		chunks[insert.Offset] = insert.Data
	}
	return inserted
}

func min64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
