package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
)

var (
	listenAddr = flag.String("l", ":7017", "listen port")
	dstAddr    = flag.String("d", "127.0.0.1:27017", "proxy to dest addr")
	isShowVer  = flag.Bool("v", false, "show version")
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 4096)
		},
	}
)

const (
	OP_REPLY                    = 1
	OP_MSG                      = 1000
	OP_UPDATE                   = 2001
	OP_INSERT                   = 2002
	OP_RESERVED                 = 2003
	OP_QUERY                    = 2004
	OP_GET_MORE                 = 2005
	OP_DELETE                   = 2006
	OP_KILL_CURSORS             = 2007
	OP_COMMAND_DEPRECATED       = 2008
	OP_COMMAND_REPLY_DEPRECATED = 2009
	OP_COMMAND                  = 2010
	OP_COMMAND_REPLY            = 2011
	OP_MSG_NEW                  = 2013
)

const version = "0.2"

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
	self.isPwClosed = true
	self.Pw.Close()
}

func (self *Parser) ParseQuery(header MsgHeader, r io.Reader) {
	flag := MustReadInt32(r)
	fullCollectionName := ReadCString(r)
	numberToSkip := MustReadInt32(r)
	numberToReturn := MustReadInt32(r)
	query := ToJson(ReadDocument(r))
	selector := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] QUERY id:%d coll:%s toskip:%d toret:%d flag:%b query:%v sel:%v\n",
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
	flag := MustReadInt32(r)
	fullCollectionName := ReadCString(r)
	docs := ReadDocuments(r)
	var docsStr string
	if len(docs) == 1 {
		docsStr = ToJson(docs[0])
	} else {
		docsStr = ToJson(docs)
	}
	fmt.Printf("%s [%s] INSERT id:%d coll:%s flag:%b docs:%v\n",
		currentTime(), self.RemoteAddr, header.RequestID, fullCollectionName, flag, docsStr)
}

func (self *Parser) ParseUpdate(header MsgHeader, r io.Reader) {
	_ = MustReadInt32(r)
	fullCollectionName := ReadCString(r)
	flag := MustReadInt32(r)
	selector := ToJson(ReadDocument(r))
	update := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] UPDATE id:%d coll:%s flag:%b sel:%v update:%v\n",
		currentTime(), self.RemoteAddr, header.RequestID, fullCollectionName, flag, selector, update)
}

func (self *Parser) ParseGetMore(header MsgHeader, r io.Reader) {
	_ = MustReadInt32(r)
	fullCollectionName := ReadCString(r)
	numberToReturn := MustReadInt32(r)
	cursorID := ReadInt64(r)
	fmt.Printf("%s [%s] GETMORE id:%d coll:%s toret:%d curID:%d\n",
		currentTime(), self.RemoteAddr, header.RequestID, fullCollectionName, numberToReturn, cursorID)
}

func (self *Parser) ParseDelete(header MsgHeader, r io.Reader) {
	_ = MustReadInt32(r)
	fullCollectionName := ReadCString(r)
	flag := MustReadInt32(r)
	selector := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] DELETE id:%d coll:%s flag:%b sel:%v \n",
		currentTime(), self.RemoteAddr, header.RequestID, fullCollectionName, flag, selector)
}

func (self *Parser) ParseKillCursors(header MsgHeader, r io.Reader) {
	_ = MustReadInt32(r)
	numberOfCursorIDs := MustReadInt32(r)
	var cursorIDs []int64
	for {
		n := ReadInt64(r)
		if n != nil {
			cursorIDs = append(cursorIDs, *n)
			continue
		}
		break
	}
	fmt.Printf("%s [%s] KILLCURSORS id:%d numCurID:%d curIDs:%d\n",
		currentTime(), self.RemoteAddr, header.RequestID, numberOfCursorIDs, cursorIDs)
}

func (self *Parser) ParseReply(header MsgHeader, r io.Reader) {
	flag := MustReadInt32(r)
	cursorID := ReadInt64(r)
	startingFrom := MustReadInt32(r)
	numberReturned := MustReadInt32(r)
	docs := ReadDocuments(r)
	var docsStr string
	if len(docs) == 1 {
		docsStr = ToJson(docs[0])
	} else {
		docsStr = ToJson(docs)
	}
	fmt.Printf("%s [%s] REPLY to:%d flag:%b curID:%d from:%d reted:%d docs:%v\n",
		currentTime(),
		self.RemoteAddr,
		header.ResponseTo,
		flag,
		cursorID,
		startingFrom,
		numberReturned,
		docsStr,
	)
}

func (self *Parser) ParseMsg(header MsgHeader, r io.Reader) {
	msg := ReadCString(r)
	fmt.Printf("%s [%s] MSG %d %s\n", currentTime(), self.RemoteAddr, header.RequestID, msg)
}
func (self *Parser) ParseReserved(header MsgHeader, r io.Reader) {
	fmt.Printf("%s [%s] RESERVED header:%v data:%v\n", currentTime(), self.RemoteAddr, header.RequestID, ToJson(header))
}

func (self *Parser) ParseCommandDeprecated(header MsgHeader, r io.Reader) {
	fmt.Printf("%s [%s] MsgHeader %v\n", currentTime(), self.RemoteAddr, ToJson(header))
	// TODO: no document, current not understand
	_, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		fmt.Printf("[%s] read failed: %v", self.RemoteAddr, err)
		return
	}
}
func (self *Parser) ParseCommandReplyDeprecated(header MsgHeader, r io.Reader) {
	fmt.Printf("%s [%s] MsgHeader %v\n", currentTime(), self.RemoteAddr, ToJson(header))
	// TODO: no document, current not understand
	_, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		fmt.Printf("[%s] read failed: %v", self.RemoteAddr, err)
		return
	}
}
func (self *Parser) ParseCommand(header MsgHeader, r io.Reader) {
	database := ReadCString(r)
	commandName := ReadCString(r)
	metadata := ToJson(ReadDocument(r))
	commandArgs := ToJson(ReadDocument(r))
	inputDocs := ToJson(ReadDocuments(r))
	fmt.Printf("%s [%s] COMMAND id:%v db:%v meta:%v cmd:%v args:%v docs %v\n",
		currentTime(),
		self.RemoteAddr,
		header.RequestID,
		database,
		metadata,
		commandName,
		commandArgs,
		inputDocs,
	)
}

func (self *Parser) ParseMsgNew(header MsgHeader, r io.Reader) {
	flags := ToJson(MustReadInt32(r))
	fmt.Printf("%s [%s] MSG start id:%v flags: %v\n",
		currentTime(),
		self.RemoteAddr,
		header.RequestID,
		flags,
	)
	for {
		t := ReadBytes(r, 1)
		if t == nil {
			fmt.Printf("%s [%s] MSG end id:%v \n",
				currentTime(),
				self.RemoteAddr,
				header.RequestID,
			)
			break
		}
		switch t[0] {
		case 0: // body
			body := ToJson(ReadDocument(r))
			checksum, _ := ReadInt32(r)
			fmt.Printf("%s [%s] MSG id:%v type:0 body: %v checksum:%v\n",
				currentTime(),
				self.RemoteAddr,
				header.RequestID,
				body,
				checksum,
			)
		case 1:
			sectionSize := MustReadInt32(r)
			r1 := io.LimitReader(r, int64(sectionSize))
			documentSequenceIdentifier := ReadCString(r1)
			objects := ToJson(ReadDocuments(r1))
			fmt.Printf("%s [%s] MSG id:%v type:1 documentSequenceIdentifier: %v objects:%v\n",
				currentTime(),
				self.RemoteAddr,
				header.RequestID,
				documentSequenceIdentifier,
				objects,
			)
		default:
			log.Panic(fmt.Sprint("unknown body kind:", t[0]))
		}
	}
}

func (self *Parser) ParseCommandReply(header MsgHeader, r io.Reader) {
	metadata := ToJson(ReadDocument(r))
	commandReply := ToJson(ReadDocument(r))
	outputDocs := ToJson(ReadDocument(r))
	fmt.Printf("%s [%s] COMMANDREPLY to:%d id:%v meta:%v cmdReply:%v outputDocs:%v\n",
		currentTime(), self.RemoteAddr, header.ResponseTo, header.RequestID, metadata, commandReply, outputDocs)
}

func (self *Parser) Parse(r *io.PipeReader) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("[%s] parser failed, painc: %v\n", self.RemoteAddr, e)
			self.isPwClosed = true
			self.Pw.Close()
		}
	}()
	for {
		header := MsgHeader{}
		err := binary.Read(r, binary.LittleEndian, &header)
		if err != nil {
			if err != io.EOF {
				log.Printf("[%s] unexpected error:%v\n", self.RemoteAddr, err)
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
		case OP_COMMAND_DEPRECATED:
			self.ParseCommandDeprecated(header, rd)
		case OP_COMMAND_REPLY_DEPRECATED:
			self.ParseCommandReplyDeprecated(header, rd)
		case OP_COMMAND:
			self.ParseCommand(header, rd)
		case OP_COMMAND_REPLY:
			self.ParseCommandReply(header, rd)
		case OP_MSG_NEW:
			self.ParseMsgNew(header, rd)
		default:
			log.Printf("[%s] unknown OpCode: %d", self.RemoteAddr, header.OpCode)
			_, err = io.Copy(ioutil.Discard, rd)
			if err != nil {
				log.Printf("[%s] read failed: %v", self.RemoteAddr, err)
				break
			}
		}
	}
}

func handleConn(conn net.Conn) {
	dst, err := net.Dial("tcp", *dstAddr)
	if err != nil {
		log.Printf("[%s] unexpected err:%v, close connection:%s\n", conn.RemoteAddr(), err, conn.RemoteAddr())
		conn.Close()
		return
	}
	defer dst.Close()
	log.Printf("[%s] new client connected: %v -> %v -> %v -> %v\n", conn.RemoteAddr(),
		conn.RemoteAddr(), conn.LocalAddr(), dst.LocalAddr(), dst.RemoteAddr())
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
	cp := func(dst io.Writer, src io.Reader, srcAddr string) {
		p := bufferPool.Get().([]byte)
		for {
			n, err := src.Read(p)
			if err != nil {
				if err != io.EOF && !isClosedErr(err) {
					log.Printf("[%s] unexpected error:%v\n", conn.RemoteAddr(), err)
				}
				log.Printf("[%s] close connection:%s\n", conn.RemoteAddr(), srcAddr)
				clean()
				break
			}
			_, err = dst.Write(p[:n])
			if err != nil {
				if err != io.EOF && !isClosedErr(err) {
					log.Printf("[%s] unexpected error:%v\n", conn.RemoteAddr(), err)
				}
				clean()
				break
			}
		}
		bufferPool.Put(p)
	}
	go cp(conn, teeReader2, dst.RemoteAddr().String())
	cp(dst, teeReader, conn.RemoteAddr().String())
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
	if *isShowVer {
		fmt.Printf("version: %s\n", version)
		os.Exit(0)
	}
	log.Printf("%s listen at %s, proxy to mongodb server %s\n", os.Args[0], *listenAddr, *dstAddr)
	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal("listen failed:", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept connection failed:", err)
			continue
		}
		go handleConn(conn)
	}
}
