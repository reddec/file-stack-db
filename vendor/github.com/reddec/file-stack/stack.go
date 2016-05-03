package fstack

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

//Stack in file
type Stack struct {
	io.Closer
	depth           int
	currentBlock    fileBlock
	currentBlockPos int64
	guard           sync.Mutex
	file            *os.File
	fileName        string
	lastAccess      time.Time
}

// Meta-info before each physical block on fs
type fileBlock struct {
	PrevBlock   uint64 // Location of prev begining of block
	HeaderPoint uint64 // Location of header begining of block
	HeaderSize  uint64 // Size in byte of header
	DataPoint   uint64 // Location of data begining of block
	DataSize    uint64 // Size in byte of data
}

// Read meta-info at specified place
func readBlockAt(reader io.ReadSeeker, offset int64) (fileBlock, error) {
	var block fileBlock
	_, err := reader.Seek(offset, os.SEEK_SET)
	if err != nil {
		return block, err
	}
	return block, binary.Read(reader, binary.LittleEndian, &block)
}

// Write meta-info to specified place
func (fb *fileBlock) writeTo(writer io.WriteSeeker, offset int64) error {
	_, err := writer.Seek(offset, os.SEEK_SET)
	if err != nil {
		return err
	}
	return binary.Write(writer, binary.LittleEndian, *fb)
}

// Calculate next block position
func (fb *fileBlock) NextBlockPoint() int64 { return int64(fb.DataPoint + fb.DataSize) }

const fileBlockDefineSize = 8 + 8 + 8 + 8 + 8

// Push header and body to stack. Returns new value of stack depth
func (s *Stack) Push(header, data []byte) (depth int, err error) {
	s.guard.Lock()
	defer s.guard.Unlock()
	s.lastAccess = time.Now()
	file, err := s.getFile()
	if err != nil {
		return -1, err
	}
	// Seek to place for next block
	currentOffset, err := file.Seek(s.currentBlock.NextBlockPoint(), os.SEEK_SET)
	if err != nil {
		return -1, err
	}
	// Get place for payload
	bodyOffset := currentOffset + fileBlockDefineSize
	block := fileBlock{
		PrevBlock:   uint64(s.currentBlockPos),
		HeaderPoint: uint64(bodyOffset),
		HeaderSize:  uint64(len(header)),
		DataPoint:   uint64(bodyOffset) + uint64(len(header)),
		DataSize:    uint64(len(data)),
	}
	// Write block meta-info
	err = binary.Write(file, binary.LittleEndian, block)
	if err != nil {
		file.Seek(currentOffset, os.SEEK_SET)
		return -1, err
	}
	// Write header
	_, err = file.Write(header)
	if err != nil {
		file.Seek(currentOffset, os.SEEK_SET)
		return -1, err
	}
	// Write data
	_, err = file.Write(data)
	if err != nil {
		file.Seek(currentOffset, os.SEEK_SET)
		return -1, err
	}
	s.depth++
	s.currentBlockPos = currentOffset
	s.currentBlock = block
	return s.depth, nil
}

// Pop one segment from tail of stack. Returns nil,nil,nil if depth is 0
func (s *Stack) Pop() (header, data []byte, err error) {
	if s.depth == 0 {
		return nil, nil, nil
	}
	s.guard.Lock()
	defer s.guard.Unlock()
	s.lastAccess = time.Now()
	file, err := s.getFile()
	if err != nil {
		return nil, nil, err
	}
	data = make([]byte, s.currentBlock.DataSize)
	header = make([]byte, s.currentBlock.HeaderSize)
	// Read header
	_, err = file.ReadAt(header, int64(s.currentBlock.HeaderPoint))
	if err != nil {
		return nil, nil, err
	}
	// Read data
	_, err = file.ReadAt(data, int64(s.currentBlock.DataPoint))
	if err != nil {
		return nil, nil, err
	}
	// Read new block if current block is not head
	var newBlock fileBlock
	if s.currentBlockPos != 0 {
		newBlock, err = readBlockAt(file, int64(s.currentBlock.PrevBlock))
		if err != nil {
			return nil, nil, err
		}
	}
	// Remove tail
	err = file.Truncate(int64(s.currentBlockPos))
	if err != nil {
		return nil, nil, err
	}
	s.depth--
	s.currentBlockPos = int64(s.currentBlock.PrevBlock)
	s.currentBlock = newBlock

	return header, data, nil
}

// Peak of stack - get one segment from stack but not remove
func (s *Stack) Peak() (header, data []byte, err error) {
	if s.depth == 0 {
		return nil, nil, nil
	}
	s.guard.Lock()
	defer s.guard.Unlock()
	s.lastAccess = time.Now()
	file, err := s.getFile()
	if err != nil {
		return nil, nil, err
	}
	data = make([]byte, s.currentBlock.DataSize)
	header = make([]byte, s.currentBlock.HeaderSize)
	// Read header
	_, err = file.ReadAt(header, int64(s.currentBlock.HeaderPoint))
	if err != nil {
		return nil, nil, err
	}
	// Read data
	_, err = file.ReadAt(data, int64(s.currentBlock.DataPoint))
	if err != nil {
		return nil, nil, err
	}
	return header, data, nil
}

// PeakHeader get only header part from tail segment from stack without remove
func (s *Stack) PeakHeader() (header []byte, err error) {
	if s.depth == 0 {
		return nil, nil
	}
	s.guard.Lock()
	defer s.guard.Unlock()
	s.lastAccess = time.Now()
	file, err := s.getFile()
	if err != nil {
		return nil, err
	}
	header = make([]byte, s.currentBlock.HeaderSize)
	// Read header
	_, err = file.ReadAt(header, int64(s.currentBlock.HeaderPoint))
	if err != nil {
		return nil, err
	}
	return header, nil
}

// Depth of stack - count of segments
func (s *Stack) Depth() int { return s.depth }

// LastAccess - time point of last access to stack
func (s *Stack) LastAccess() time.Time { return s.lastAccess }

// IterateBackward - iterate over hole stack segment-by-segment from end to begining
func (s *Stack) IterateBackward(handler func(depth int, header io.Reader, body io.Reader) bool) error {
	s.guard.Lock()
	defer s.guard.Unlock()
	if s.depth == 0 {
		return nil
	}
	file, err := s.getFile()
	if err != nil {
		return err
	}
	defer file.Seek(0, os.SEEK_END)
	var (
		currentBlock       fileBlock // Current block description
		currentBlockOffset uint64    // Current block offset from begining of file
	)
	currentBlock = s.currentBlock
	currentBlockOffset = uint64(s.currentBlockPos)
	depth := s.depth
	for {
		body := io.NewSectionReader(file, int64(currentBlock.DataPoint), int64(currentBlock.DataSize))
		header := io.NewSectionReader(file, int64(currentBlock.HeaderPoint), int64(currentBlock.HeaderSize))
		// invoke block processor
		if handler != nil && !handler(depth, header, body) {
			return nil
		}
		if currentBlock.PrevBlock > currentBlockOffset {
			log.Printf("Danger back-ref link: prev block %v has greater index then current %v", currentBlock.PrevBlock, currentBlockOffset)
		}

		depth--
		if currentBlock.PrevBlock == currentBlockOffset {
			// First block has prev block = 0
			break
		}
		currentBlockOffset = currentBlock.PrevBlock
		currentBlock, err = readBlockAt(file, int64(currentBlock.PrevBlock))
		if err != nil {
			return err
		}
	}
	if depth != 0 {
		log.Println("Broker back path detected at", depth, "depth index")
	}
	return nil
}

// IterateForward - iterate over hole stack segment-by-segment from begining to end. If all segments
// iterated stack may be repaired
func (s *Stack) IterateForward(handler func(depth int, header io.Reader, body io.Reader) bool) error {
	// This operation does not relies on depth counter, so can be used for repare
	s.guard.Lock()
	defer s.guard.Unlock()
	s.lastAccess = time.Now()
	file, err := s.getFile()
	if err != nil {
		return err
	}
	//Get file size
	fileSize, err := file.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}
	defer file.Seek(0, os.SEEK_END)
	var (
		currentBlock       fileBlock // Current block description
		currentBlockOffset uint64    // Current block offset from begining of file
	)
	var depth int
	for currentBlock.NextBlockPoint() < fileSize {
		newPos := currentBlock.NextBlockPoint()
		if newPos > fileSize {
			log.Println("Bad reference to next block at", currentBlockOffset, "!trunc!")
			file.Truncate(int64(currentBlockOffset))
			break
		}
		block, err := readBlockAt(file, newPos)
		// Non-full meta-info?
		if err == io.EOF {
			log.Println("Broken meta info at", newPos, "!trunc!")
			file.Truncate(newPos)
			break
		}
		// I/O error
		if err != nil {
			log.Println("Can't read block at", newPos)
			return err
		}
		// Check back-ref
		if block.PrevBlock != currentBlockOffset {
			log.Println("Bad back reference", block.PrevBlock, "!=", currentBlockOffset, "!upd!")
			block.PrevBlock = currentBlockOffset
			block.writeTo(file, newPos)
		}
		// Update current state
		currentBlockOffset = uint64(newPos)
		currentBlock = block
		body := io.NewSectionReader(file, int64(currentBlock.DataPoint), int64(currentBlock.DataSize))
		header := io.NewSectionReader(file, int64(currentBlock.HeaderPoint), int64(currentBlock.HeaderSize))
		// invoke block processor
		if handler != nil && !handler(depth, header, body) {
			return nil
		}
		depth++
	}
	s.depth = depth
	s.currentBlock = currentBlock
	s.currentBlockPos = int64(currentBlockOffset)
	return nil
}

// Repare stack segements
func (s *Stack) Repare() error { return s.IterateForward(nil) }

// Close backend stack file. If access is requried, file will automatically reopened
func (s *Stack) Close() error {
	s.guard.Lock()
	defer s.guard.Unlock()
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

func (s *Stack) getFile() (*os.File, error) {
	if s.file == nil {
		f, err := os.OpenFile(s.fileName, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			return nil, err
		}
		s.file = f
	}
	return s.file, nil
}

// OpenStack - open or create stack
func OpenStack(filename string) (*Stack, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return nil, err
	}
	return NewStack(file)
}

// CreateStack - create or truncate stack
func CreateStack(filename string) (*Stack, error) {
	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		return nil, err
	}
	return NewStack(file)
}

// NewStack - create new stack based on file
func NewStack(file *os.File) (*Stack, error) {
	stack := &Stack{file: file, fileName: file.Name()}
	err := stack.Repare()
	if err != nil {
		stack.Close()
		return nil, err
	}
	return stack, nil
}
