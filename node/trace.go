package node

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"sync"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pb "github.com/libp2p/go-libp2p-pubsub/pb"
)

type TraceCollector struct {
	host      host.Host
	dir       string
	jsonTrace string

	mx  sync.Mutex
	buf []*pb.TraceEvent

	notifyWriteCh chan struct{}
	flushFileCh   chan struct{}
	exitCh        chan struct{}
	doneCh        chan struct{}
}

// NewTraceCollector creates a new pubsub traces collector. A collector is a process
// that listens on a libp2p endpoint, accepts pubsub tracing streams from peers,
// and records the incoming data into rotating gzip files.
// If the json argument is not empty, then every time a new trace is generated, it will be written
// to this directory in json format for online processing.
func NewTraceCollector(host host.Host, dir, jsonTrace string) (*TraceCollector, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}

	c := &TraceCollector{
		host:          host,
		dir:           dir,
		jsonTrace:     jsonTrace,
		notifyWriteCh: make(chan struct{}, 1),
		flushFileCh:   make(chan struct{}, 1),
		exitCh:        make(chan struct{}, 1),
		doneCh:        make(chan struct{}, 1),
	}

	host.SetStreamHandler(pubsub.RemoteTracerProtoID, c.handleStream)

	// go c.collectWorker()
	// go c.monitorWorker()

	return c, nil
}

// Stop stops the collector.
func (tc *TraceCollector) Stop() {
	close(tc.exitCh)
	tc.flushFileCh <- struct{}{}
	<-tc.doneCh
}

// Flush flushes and rotates the current file.
func (tc *TraceCollector) Flush() {
	tc.flushFileCh <- struct{}{}
}

// handleStream accepts an incoming tracing stream and drains it into the
// buffer, until the stream is closed or an error occurs.
func (tc *TraceCollector) handleStream(s network.Stream) {
	defer s.Close()

	fmt.Printf("new stream from", s.Conn().RemotePeer())

	gzipR, err := gzip.NewReader(s)
	if err != nil {
		fmt.Printf("error opening compressed stream from %s: %s\n", s.Conn().RemotePeer(), err)
		s.Reset()
		return
	}

	r := ggio.NewDelimitedReader(gzipR, 1<<22)
	var msg pb.TraceEventBatch

	for {
		msg.Reset()

		switch err = r.ReadMsg(&msg); err {
		case nil:
			tc.mx.Lock()
			tc.buf = append(tc.buf, msg.Batch...)
			tc.mx.Unlock()

			select {
			case tc.notifyWriteCh <- struct{}{}:
			default:
			}

		case io.EOF:
			return

		default:
			fmt.Sprintf("error reading batch from %s: %s\n", s.Conn().RemotePeer(), err)
			return
		}
	}
}

// // collectWorker is the main worker. It keeps recording traces into the
// // `current` file and rotates the file when it's filled.
// func (tc *TraceCollector) collectWorker() {
// 	defer close(tc.doneCh)

// 	current := fmt.Sprintf("%s/current", tc.dir)

// 	for {
// 		out, err := os.OpenFile(current, os.O_CREATE|os.O_WRONLY, 0644)
// 		if err != nil {
// 			panic(err)
// 		}

// 		err = tc.writeFile(out)
// 		if err != nil {
// 			panic(err)
// 		}

// 		// Rotate the file.
// 		base := fmt.Sprintf("trace.%d", time.Now().UnixNano())
// 		next := fmt.Sprintf("%s/%s.pb.gz", tc.dir, base)
// 		logger.Debugf("move %s -> %s", current, next)
// 		err = os.Rename(current, next)
// 		if err != nil {
// 			panic(err)
// 		}

// 		// Generate the json output if so desired
// 		if tc.jsonTrace != "" {
// 			tc.writeJsonTrace(next, base)
// 		}

// 		// yield if we're done.
// 		select {
// 		case <-tc.exitCh:
// 			return
// 		default:
// 		}
// 	}
// }

// func (tc *TraceCollector) writeJsonTrace(trace, name string) {
// 	// open the trace, read it and transcode to json
// 	in, err := os.Open(trace)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer in.Close()

// 	gzipR, err := gzip.NewReader(in)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer gzipR.Close()

// 	tmpTrace := fmt.Sprintf("/tmp/%s.json", name)
// 	out, err := os.OpenFile(tmpTrace, os.O_WRONLY|os.O_CREATE, 0644)
// 	if err != nil {
// 		panic(err)
// 	}

// 	var evt pb.TraceEvent
// 	pbr := ggio.NewDelimitedReader(gzipR, 1<<20)
// 	enc := json.NewEncoder(out)
// loop:
// 	for {
// 		evt.Reset()

// 		switch err = pbr.ReadMsg(&evt); err {
// 		case nil:
// 			err = enc.Encode(&evt)
// 			if err != nil {
// 				panic(err)
// 			}
// 		case io.EOF:
// 			break loop
// 		default:
// 			panic(err)
// 		}
// 	}

// 	err = out.Close()
// 	if err != nil {
// 		panic(err)
// 	}

// 	jsonTrace := fmt.Sprintf("%s/%s.json", tc.jsonTrace, name)
// 	err = os.Rename(tmpTrace, jsonTrace)
// 	if err != nil {
// 		panic(err)
// 	}
// }

// // monitorWorker watches the current file and triggers closure+rotation based on
// // time and size.
// //
// // TODO (#3): this needs to use select so we can rcv from exitCh and yield.
// func (tc *TraceCollector) monitorWorker() {
// 	current := fmt.Sprintf("%s/current", tc.dir)

// Outer:
// 	for {
// 		start := time.Now()

// 	Inner:
// 		for {
// 			time.Sleep(time.Minute)

// 			now := time.Now()
// 			if now.After(start.Add(MaxLogTime)) {
// 				select {
// 				case tc.flushFileCh <- struct{}{}:
// 				default:
// 				}
// 				continue Outer
// 			}

// 			finfo, err := os.Stat(current)
// 			if err != nil {
// 				logger.Warningf("error stating trace log file: %s", err)
// 				continue Inner
// 			}

// 			if finfo.Size() > int64(MaxLogSize) {
// 				select {
// 				case tc.flushFileCh <- struct{}{}:
// 				default:
// 				}

// 				continue Outer
// 			}
// 		}
// 	}
// }
