/*
The distributed storage implemented in this package requires fix sized chunks of content
Chunker is the interface to a component that is responsible for disassembling and assembling larger data.

TreeChunker implements a Chunker based on a tree structure defined as follows:

1 each node in the tree including the root and other branching nodes are stored as a chunk.

2 branching nodes encode data contents that includes the size of the dataslice covered by its entire subtree under the node as well as the hash keys of all its children :
data_{i} := size(subtree_{i}) || key_{j} || key_{j+1} .... || key_{j+n-1}

3 Leaf nodes encode an actual subslice of the input data.

4 if data size is not more than maximum chunksize, the data is stored in a single chunk
  key = sha256(int64(size) + data)

2 if data size is more than chunksize*Branches^l, but no more than chunksize*
  Branches^(l+1), the data vector is split into slices of chunksize*
  Branches^l length (except the last one).
  key = sha256(int64(size) + key(slice0) + key(slice1) + ...)
*/

package bzz

import (
	"crypto"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	hasherfunc      crypto.Hash = crypto.SHA256 // http://golang.org/pkg/hash/#Hash
	defaultBranches int64       = 128
)

var (
	// hashSize     int64 = hasherfunc.New().Size() // hasher knows about its own length in bytes
	// chunksize    int64 = branches * hashSize     // chunk is defined as this
	joinTimeout  = 120 * time.Second
	splitTimeout = 120 * time.Second
)

/*
Chunker is the interface to a component that is responsible for disassembling and assembling larger data and indended to be the dependency of a DPA storage system with fixed maximum chunksize.
It relies on the underlying chunking model.
When calling Split, the caller provides a channel (chan *Chunk) on which it receives chunks to store. The DPA delegates to storage layers (implementing ChunkStore interface). NewChunkstore(DB) is a convenience wrapper with which all DBs (conforming to DB interface) can serve as ChunkStores. See chunkStore.go
After getting notified that all the data has been split (the error channel is closed), the caller can safely read or save the root key. Optionally it times out if not all chunks get stored or not the entire stream of data has been processed. By inspecting the errc channel the caller can check if any explicit errors (typically IO read/write failures) occured during splitting.

When calling Join with a root key, the caller gets returned a lazy reader. The caller again provides a channel and receives an error channel. The chunk channel is the one on which the caller receives placeholder chunks with missing data. The DPA is supposed to forward this to the chunk stores and notify the chunker if the data has been delivered (i.e. retrieved from memory cache, disk-persisted db or cloud based swarm delivery). The chunker then puts these together and notifies the DPA if data has been assembled by a closed error channel. Once the DPA finds the data has been joined, it is free to deliver it back to swarm in full (if the original request was via the bzz protocol) or save and serve if it it was a local client request.

*/
type Chunker interface {
	/*
	 	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	 	New chunks to store are coming to caller via the chunk storage channel, which the caller provides.
	 	wg is a Waitgroup (can be nil) that can be used to block until the local storage finishes
	 	The caller gets returned an error channel, if an error is encountered during splitting, it is fed to errC error channel.
	   A closed error signals process completion at which point the key can be considered final if there were no errors.
	*/
	Split(key *common.Hash, data SectionReader, chunkC chan *Chunk, wg *sync.WaitGroup) chan error
	/*
		Join reconstructs original content based on a root key.
		When joining, the caller gets returned a Lazy SectionReader
		New chunks to retrieve are coming to caller via the Chunk channel, which the caller provides.
		If an error is encountered during joining, it appears as a reader error.
		The SectionReader provides on-demand fetching of chunks.
	*/
	Join(key *common.Hash, chunkC chan *Chunk) SectionReader

	// returns the key length
	KeySize() int64
}

/*
Tree chunker is a concrete implementation of data chunking.
This chunker works in a simple way, it builds a tree out of the document so that each node either represents a chunk of real data or a chunk of data representing an branching non-leaf node of the tree. In particular each such non-leaf chunk will represent is a concatenation of the hash of its respective children. This scheme simultaneously guarantees data integrity as well as self addressing. Abstract nodes are transparent since their represented size component is strictly greater than their maximum data size, since they encode a subtree.

If all is well it is possible to implement this by simply composing readers so that no extra allocation or buffering is necessary for the data splitting and joining. This means that in principle there can be direct IO between : memory, file system, network socket (bzz peers storage request is read from the socket ). In practice there may be need for several stages of internal buffering.
Unfortunately the hashing itself does use extra copies and allocation though since it does need it.
*/

type TreeChunker struct {
	Branches     int64
	HashFunc     crypto.Hash
	JoinTimeout  time.Duration
	SplitTimeout time.Duration
	// calculated
	hashSize  int64 // self.HashFunc.New().Size()
	chunkSize int64 // hashSize* Branches
}

func (self *TreeChunker) Init() {
	if self.HashFunc == 0 {
		self.HashFunc = hasherfunc
	}
	if self.Branches == 0 {
		self.Branches = defaultBranches
	}
	if self.JoinTimeout == 0 {
		self.JoinTimeout = joinTimeout
	}
	if self.SplitTimeout == 0 {
		self.SplitTimeout = splitTimeout
	}
	self.hashSize = int64(self.HashFunc.New().Size())
	self.chunkSize = self.hashSize * self.Branches
	// dpaLogger.Debugf("Chunker initialised: branches: %v, hashsize: %v, chunksize: %v, join timeout: %v , split timeout: %v", self.Branches, self.hashSize, self.chunkSize, self.JoinTimeout, self.SplitTimeout)

}

func (self *TreeChunker) KeySize() int64 {
	return self.hashSize
}

// String() for pretty printing
func (self *Chunk) String() string {
	var size int64
	var n int
	return fmt.Sprintf("Key: [%x..] TreeSize: %v Chunksize: %v Data: %x\n", self.Key[:4], self.Size, size, self.SData[:n])
}

// The treeChunkers own Hash hashes together
// - the size (of the subtree encoded in the Chunk)
// - the Chunk, ie. the contents read from the input reader
func (self *TreeChunker) Hash(input []byte) []byte {
	hasher := self.HashFunc.New()
	hasher.Write(input)
	return hasher.Sum(nil)
}

func (self *TreeChunker) Split(key *common.Hash, data SectionReader, chunkC chan *Chunk, swg *sync.WaitGroup) (errC chan error) {

	if swg != nil {
		swg.Add(1)
		defer swg.Done()
	}

	if self.chunkSize <= 0 {
		panic("chunker must be initialised")
	}

	wg := &sync.WaitGroup{}
	errC = make(chan error)
	rerrC := make(chan error)
	timeout := time.After(self.SplitTimeout)
	// dpaLogger.Debugf("split request received")
	// fmt.Printf("split request received (chunk size %v)\n", self.chunkSize)

	wg.Add(1)
	go func() {

		depth := 0
		treeSize := self.chunkSize
		size := data.Size()
		// takes lowest depth such that chunksize*HashCount^(depth+1) > size
		// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.

		for ; treeSize < size; treeSize *= self.Branches {
			depth++
		}

		// dpaLogger.Debugf("split request received for data (%v bytes, depth: %v)", size, depth)
		// fmt.Printf("split request received for data (%v bytes, depth: %v)\n", size, depth)

		//launch actual recursive function passing the workgroup
		self.split(depth, treeSize/self.Branches, key, data, chunkC, rerrC, wg, swg)
		// fmt.Printf("split request d for data  %x\n", key)

	}()

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {
		wg.Wait()
		close(rerrC)
		// fmt.Printf("split request d for data  %x\n", key)

	}()

	// waiting for request to end with wg finishing, error, or timeout
	go func() {
		select {
		case err := <-rerrC:
			if err != nil {
				errC <- err
			} // otherwise splitting is complete
		case <-timeout:
			errC <- fmt.Errorf("split time out")
		}
		close(errC)
	}()

	return
}

func (self *TreeChunker) split(depth int, treeSize int64, key *common.Hash, data SectionReader, chunkC chan *Chunk, errc chan error, parentWg *sync.WaitGroup, swg *sync.WaitGroup) {

	defer parentWg.Done()

	size := data.Size()
	var newChunk *Chunk
	var hash = &common.Hash{}
	// dpaLogger.Debugf("depth: %v, max subtree size: %v, data size: %v", depth, treeSize, size)
	// fmt.Printf("depth: %v, max subtree size: %v, data size: %v\n", depth, treeSize, size)

	for depth > 0 && size < treeSize {
		treeSize /= self.Branches
		depth--
	}

	if depth == 0 {
		// leaf nodes -> content chunks
		chunkData := make([]byte, data.Size()+8)
		binary.LittleEndian.PutUint64(chunkData[0:8], uint64(size))
		data.ReadAt(chunkData[8:], 0)
		copy(hash[:], self.Hash(chunkData))
		// dpaLogger.Debugf("content chunk: max subtree size: %v, data size: %v", treeSize, size)
		// fmt.Printf("content chunk: max subtree size: %v, data size: %v, chunkData: %x, hash: %x\n", treeSize, size, chunkData, hash)
		newChunk = &Chunk{
			Key:   hash,
			SData: chunkData,
			Size:  size,
		}
	} else {
		// intermediate chunk containing child nodes hashes
		branchCnt := int64((size + treeSize - 1) / treeSize)
		// dpaLogger.Debugf("intermediate node: setting branches: %v, depth: %v, max subtree size: %v, data size: %v", branchCnt, depth, treeSize, size)

		var chunk []byte = make([]byte, branchCnt*self.hashSize+8)
		var pos, i int64

		binary.LittleEndian.PutUint64(chunk[0:8], uint64(size))

		childrenWg := &sync.WaitGroup{}
		var secSize int64
		for i < branchCnt {
			// the last item can have shorter data
			if size-pos < treeSize {
				secSize = size - pos
			} else {
				secSize = treeSize
			}
			// take the section of the data corresponding encoded in the subTree
			subTreeData := NewChunkReader(data, pos, secSize)
			// the hash of that data
			subTreeKey := &common.Hash{}
			copy(subTreeKey[:], chunk[8+i*self.hashSize:8+(i+1)*self.hashSize])

			childrenWg.Add(1)
			go self.split(depth-1, treeSize/self.Branches, subTreeKey, subTreeData, chunkC, errc, childrenWg, swg)

			i++
			pos += treeSize
		}
		// wait for all the children to complete calculating their hashes and copying them onto sections of the chunk
		childrenWg.Wait()
		// now we got the hashes in the chunk, then hash the chunk
		/*		chunkReader := NewChunkReaderFromBytes(chunk) // bytes.Reader almost implements SectionReader
				chunkData := make([]byte, chunkReader.Size())
				chunkReader.ReadAt(chunkData, 0)*/

		copy(hash[:], self.Hash(chunk))
		newChunk = &Chunk{
			Key:   hash,
			SData: chunk,
			Size:  size,
			wg:    swg,
		}

		if swg != nil {
			swg.Add(1)
		}
	}

	// send off new chunk to storage
	if chunkC != nil {
		chunkC <- newChunk
	}
	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)

	copy(key[:], hash[:])
	// fmt.Printf("key: %x, hash: %x\n", key, hash)

}

func (self *TreeChunker) Join(key *common.Hash, chunkC chan *Chunk) SectionReader {

	return &LazyChunkReader{
		key:     key,
		chunkC:  chunkC,
		quitC:   make(chan bool),
		errC:    make(chan error),
		chunker: self,
	}
}

// LazyChunkReader implements LazySectionReader
type LazyChunkReader struct {
	key     *common.Hash
	chunkC  chan *Chunk
	size    int64
	off     int64
	quitC   chan bool
	errC    chan error
	chunker *TreeChunker
}

func (self *LazyChunkReader) ReadAt(b []byte, off int64) (read int, err error) {
	self.errC = make(chan error)
	chunk := &Chunk{
		Key: self.key,
		C:   make(chan bool), // close channel to signal data delivery
	}
	self.chunkC <- chunk // submit retrieval request, someone should be listening on the other side (or we will time out globally)
	// dpaLogger.Debugf("readAt %x into %d bytes at %d.", chunk.Key[:4], len(b), off)

	// waiting for the chunk retrieval
	select {
	case <-self.quitC:
		// this is how we control process leakage (quitC is closed once join is finished (after timeout))
		// dpaLogger.Debugf("quit")
		return
	case <-chunk.C: // bells are ringing, data have been delivered
		// dpaLogger.Debugf("chunk data received for %x", chunk.Key[:4])
	}
	if len(chunk.SData) == 0 {
		// dpaLogger.Debugf("No payload in %x.", chunk.Key)
		return 0, notFound
	}
	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
	self.size = chunk.Size
	if b == nil {
		// dpaLogger.Debugf("Size query for %x.", chunk.Key[:4])
		return
	}
	want := int64(len(b))
	if off+want > self.size {
		want = self.size - off
	}
	var treeSize int64
	var depth int
	// calculate depth and max treeSize
	treeSize = self.chunker.chunkSize
	for ; treeSize < chunk.Size; treeSize *= self.chunker.Branches {
		depth++
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go self.join(b, off, off+want, depth, treeSize/self.chunker.Branches, chunk, &wg)
	go func() {
		wg.Wait()
		close(self.errC)
	}()
	select {
	case err = <-self.errC:
		// dpaLogger.Debugf("ReadAt received %v.", err)
		read = len(b)
		if off+int64(read) == self.size {
			err = io.EOF
		}
		// dpaLogger.Debugf("ReadAt returning with %d, %v.", read, err)
	case <-self.quitC:
		// dpaLogger.Debugf("ReadAt aborted with %d, %v.", read, err)
	}
	return
}

func (self *LazyChunkReader) join(b []byte, off int64, eoff int64, depth int, treeSize int64, chunk *Chunk, parentWg *sync.WaitGroup) {
	defer parentWg.Done()

	// dpaLogger.Debugf("depth: %v, loff: %v, eoff: %v, chunk.Size: %v, treeSize: %v", depth, off, eoff, chunk.Size, treeSize)

	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	// find appropriate block level
	for chunk.Size < treeSize && depth > 0 {
		treeSize /= self.chunker.Branches
		depth--
	}

	if depth == 0 {
		// dpaLogger.Debugf("len(b): %v, off: %v, eoff: %v, chunk.Size: %v, treeSize: %v", depth, len(b), off, eoff, chunk.Size, treeSize)
		if int64(len(b)) != eoff-off {
			//fmt.Printf("len(b) = %v  off = %v  eoff = %v", len(b), off, eoff)
			panic("len(b) does not match")
		}

		copy(b, chunk.SData[8+off:8+eoff])
		return // simply give back the chunks reader for content chunks
	}

	// subtree index
	start := off / treeSize
	end := (eoff + treeSize - 1) / treeSize
	wg := sync.WaitGroup{}

	for i := start; i < end; i++ {

		soff := i * treeSize
		roff := soff
		seoff := soff + treeSize

		if soff < off {
			soff = off
		}
		if seoff > eoff {
			seoff = eoff
		}

		wg.Add(1)
		go func(j int64) {
			childKey := &common.Hash{}
			copy(childKey[:], chunk.SData[8+j*self.chunker.hashSize:8+(j+1)*self.chunker.hashSize])
			// dpaLogger.Debugf("subtree index: %v -> %x", j, childKey[:4])

			ch := &Chunk{
				Key: childKey,
				C:   make(chan bool), // close channel to signal data delivery
			}
			// dpaLogger.Debugf("chunk data sent for %x (key interval in chunk %v-%v)", ch.Key[:4], j*self.chunker.hashSize, (j+1)*self.chunker.hashSize)
			self.chunkC <- ch // submit retrieval request, someone should be listening on the other side (or we will time out globally)

			// waiting for the chunk retrieval
			select {
			case <-self.quitC:
				// this is how we control process leakage (quitC is closed once join is finished (after timeout))
				return
			case <-ch.C: // bells are ringing, data have been delivered
				// dpaLogger.Debugf("chunk data received")
			}
			if soff < off {
				soff = off
			}
			if len(ch.SData) == 0 {
				self.errC <- fmt.Errorf("chunk %v-%v not found", off, off+treeSize)
				return
			}
			self.join(b[soff-off:seoff-off], soff-roff, seoff-roff, depth-1, treeSize/self.chunker.Branches, ch, &wg)
		}(i)
	} //for
	wg.Wait()
}
