package indexer

import "errors"

type Chunk struct{ 
	content string 
	index int
}

// TODO v0 Chunking strategy is simple fixed length chunking
func ChunkText(text string, chunkSize int) ([]Chunk, error){

	if chunkSize <= 0 {
		return nil, errors.New("chunk size must be greater than zero")
	}

	// Chunk by rune, not by byte, so a chunk boundary never lands in the
	// middle of a multi-byte UTF-8 codepoint.
	runes := []rune(text)

	var idx int
	var chunks []Chunk

	for i := 0; i < len(runes); i += chunkSize {
		end := min(i+chunkSize, len(runes))
		chunks = append(chunks, Chunk{index: idx, content: string(runes[i:end])})
		idx += 1
	}

	return chunks, nil
}


func ChunkFile(path string, chunkSize int) ([]Chunk, error) {

	content, err := ReadFile(path)

	if err != nil {
		return nil, err
	}

	res, err := ChunkText(content, chunkSize)

	if err != nil{
		return nil, err
	}

	return res, nil
}

