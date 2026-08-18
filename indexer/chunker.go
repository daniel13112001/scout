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

	var i int
	var idx int

	var chunks []Chunk


	for i < len(text){
		end := i + chunkSize
		if (end > len(text)){
			end = len(text)
		}
		ch := Chunk{index: idx, content: text[i:end]}
		chunks = append(chunks, ch)
		i += chunkSize
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

