package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

var (
	listenAddr = flag.String("l", ":7017", "listen port")
	dstAddr    = flag.String("d", "127.0.0.1:27017", "proxy to dest addr")
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}
)

const (
	OP_REPLY        = 1
	OP_MSG          = 1000
	OP_UPDATE       = 2001
	OP_INSERT       = 2002
	OP_RESERVED     = 2003
	OP_QUERY        = 2004
	OP_GET_MORE     = 2005
	OP_DELETE       = 2006
	OP_KILL_CURSORS = 2007
)

type MsgHeader struct {
	MessageLength int32
	RequestID     int32
	ResponseTo    int32
	OpCode        int32
}

type Parser struct {
	Pw         *io.PipeWriter
	RemoteAddr string
	isPwClosed bool
}

func NewParser(remoteAddr string) *Parser {
	pr, pw := io.Pipe()
	parser := &Parser{
		Pw:         pw,
		RemoteAddr: remoteAddr,
	}
	go parser.Parse(pr)
	return parser
}

func (self *Parser) Write(p []byte) (n int, err error) {
	if self.isPwClosed {
		return len(p), nil
	}
	return self.Pw.Write(p)
}

func (self *Parser) Close() {
	self.Pw.Close()
	self.isPwClosed = true
}

func (self *Parser) ParseQuery(header MsgHeader, r io.Reader) {
	flag := ReadInt32(r)
	fullCollectionName := ReadCString(r)
	numberToSkip := ReadInt32(r)
	numberToReturn := ReadInt32(r)
	query := ToJson(ReadDocument(r))
	selector := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] QUERY id:%d ns:%s skip:%d ret:%d flag:%b query:%v sel:%v\n",
		currentTime(),
		self.RemoteAddr,
		header.RequestID,
		fullCollectionName,
		numberToSkip,
		numberToReturn,
		flag,
		query,
		selector,
	)
}

func (self *Parser) ParseInsert(header MsgHeader, r io.Reader) {
	flag := ReadInt32(r)
	ns := ReadCString(r)
	docs := ToJson(ReadDocuments(r))
	fmt.Printf("%s [%s] INSERT id:%d ns:%s flag:%b docs:%v\n",
		currentTime(), self.RemoteAddr, header.RequestID, ns, flag, docs)
}

func (self *Parser) ParseUpdate(header MsgHeader, r io.Reader) {
	_ = ReadInt32(r)
	ns := ReadCString(r)
	flag := ReadInt32(r)
	selector := ToJson(ReadDocument(r))
	update := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] UPDATE id:%d ns:%s flag:%b sel:%v update:%v\n",
		currentTime(), self.RemoteAddr, header.RequestID, ns, flag, selector, update)
}

func (self *Parser) ParseGetMore(header MsgHeader, r io.Reader) {
	_ = ReadInt32(r)
	ns := ReadCString(r)
	numberToReturn := ReadInt32(r)
	cursorID := ReadInt64(r)
	fmt.Printf("%s [%s] GETMORE id:%d ns:%s ret:%d curID:%d\n",
		currentTime(), self.RemoteAddr, header.RequestID, ns, numberToReturn, cursorID)
}

func (self *Parser) ParseDelete(header MsgHeader, r io.Reader) {
	_ = ReadInt32(r)
	ns := ReadCString(r)
	flag := ReadInt32(r)
	selector := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] DELETE id:%d ns:%s flag:%b sel:%v \n",
		currentTime(), self.RemoteAddr, header.RequestID, ns, flag, selector)
}

func (self *Parser) ParseKillCursors(header MsgHeader, r io.Reader) {
	_ = ReadInt32(r)
	numberOfCursorIDs := ReadInt32(r)
	// todo array
	cursorIDs := ReadInt64(r)
	fmt.Printf("%s [%s] KILLCURSORS id:%d numCurID:%d curIDs:%d\n",
		currentTime(), self.RemoteAddr, header.RequestID, numberOfCursorIDs, cursorIDs)
}

func (self *Parser) ParseReply(header MsgHeader, r io.Reader) {
	flag := ReadInt32(r)
	cursorID := ReadInt64(r)
	startingFrom := ReadInt32(r)
	numberReturned := ReadInt32(r)
	documents := ToJson(ReadDocuments(r))
	fmt.Printf("%s [%s] REPLY to:%d flag:%b curID:%d from:%d ret:%d docs:%v\n",
		currentTime(),
		self.RemoteAddr,
		header.ResponseTo,
		flag,
		cursorID,
		startingFrom,
		numberReturned,
		documents,
	)
}

func (self *Parser) ParseMsg(header MsgHeader, r io.Reader) {
	msg := ReadCString(r)
	fmt.Printf("%s [%s] MSG %d %s\n", currentTime(), header.RequestID, msg)
}
func (self *Parser) ParseReserved(header MsgHeader, r io.Reader) {
	fmt.Printf("%s [%s] RESERVED header:%v data:%s\n", currentTime(), header.RequestID, header)
}

func (self *Parser) Parse(r *io.PipeReader) {
	defer func() {
		if e := recover(); e != nil {
			log.Println("parser failed, painc:", e)
			self.isPwClosed = true
			self.Pw.Close()
			r.Close()
		}
	}()
	for {
		header := MsgHeader{}
		err := binary.Read(r, binary.LittleEndian, &header)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			break
		}
		rd := io.LimitReader(r, int64(header.MessageLength-4*4))
		switch header.OpCode {
		case OP_QUERY:
			self.ParseQuery(header, rd)
		case OP_INSERT:
			self.ParseInsert(header, rd)
		case OP_DELETE:
			self.ParseDelete(header, rd)
		case OP_UPDATE:
			self.ParseUpdate(header, rd)
		case OP_MSG:
			self.ParseMsg(header, rd)
		case OP_REPLY:
			self.ParseReply(header, rd)
		case OP_GET_MORE:
			self.ParseGetMore(header, rd)
		case OP_KILL_CURSORS:
			self.ParseKillCursors(header, rd)
		case OP_RESERVED:
			self.ParseReserved(header, rd)
		}
	}
}

func handleConn(conn net.Conn) {
	log.Println("new client connected from:", conn.RemoteAddr())
	dst, err := net.Dial("tcp", *dstAddr)
	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}
	defer dst.Close()
	parser := NewParser(conn.RemoteAddr().String())
	parser2 := NewParser(conn.RemoteAddr().String())
	teeReader := io.TeeReader(conn, parser)
	teeReader2 := io.TeeReader(dst, parser2)
	clean := func() {
		conn.Close()
		dst.Close()
		parser.Close()
		parser2.Close()
	}
	cp := func(dst io.Writer, src io.Reader) {
		p := bufferPool.Get().([]byte)
		for {
			n, err := src.Read(p)
			if err != nil {
				if err != io.EOF && !isClosedErr(err) {
					log.Println(err)
				}
				clean()
				break
			}
			_, err = dst.Write(p[:n])
			if err != nil {
				if err != io.EOF && !isClosedErr(err) {
					log.Println(err)
				}
				clean()
				break
			}
		}
		bufferPool.Put(p)
	}
	go cp(conn, teeReader2)
	cp(dst, teeReader)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
	log.Printf("%s listen at :%s and proxy to %s\n", os.Args[0], *listenAddr, *dstAddr)
	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConn(conn)
	}
}
