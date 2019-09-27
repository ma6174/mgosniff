package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/globalsign/mgo/bson"
)

func convertToBin(n uint32) (s string) {
	s = strconv.FormatUint(uint64(n), 2)
	for i := len(s); i < 32; i++ {
		s = "0" + s
	}
	return
}

func MustReadUint32(r io.Reader) (n uint32) {
	err := binary.Read(r, binary.LittleEndian, &n)
	if err != nil {
		panic(err)
	}
	return
}

func MustReadInt32(r io.Reader) (n int32) {
	err := binary.Read(r, binary.LittleEndian, &n)
	if err != nil {
		panic(err)
	}
	return
}
func ReadInt32(r io.Reader) (n int32, err error) {
	err = binary.Read(r, binary.LittleEndian, &n)
	return
}

func ReadInt64(r io.Reader) *int64 {
	var n int64
	err := binary.Read(r, binary.LittleEndian, &n)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		panic(err)
	}
	return &n
}

func ReadBytes(r io.Reader, n int) []byte {
	b := make([]byte, n)
	_, err := r.Read(b)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		panic(err)
	}
	return b
}

func ReadCString(r io.Reader) string {
	var b []byte
	var one = make([]byte, 1)
	for {
		_, err := r.Read(one)
		if err != nil {
			panic(err)
		}
		if one[0] == '\x00' {
			break
		}
		b = append(b, one[0])
	}
	return string(b)
}

func ReadOne(r io.Reader) (nBytes int32, buf []byte) {
	docLen, err := ReadInt32(r)
	fmt.Println("docLen: ", docLen)
	if err != nil {
		if err == io.EOF {
			return 0, nil
		}
		panic(err)
	}

	buf = make([]byte, int(docLen))
	binary.LittleEndian.PutUint32(buf, uint32(docLen))
	nb, err := io.ReadFull(r, buf[4:])
	if err != nil {
		panic(err)
	}

	nBytes = int32(nb+4)

	return nBytes, buf
}

func ReadDocumentSz(r io.Reader) (nBytes int32, m bson.M) {
	if nb, one := ReadOne(r); one != nil {
		err := bson.Unmarshal(one, &m)
		if err != nil {
			panic(err)
		}
		nBytes = nb
	}

	return
}

func ReadDocument(r io.Reader) (m bson.M) {
	_, m = ReadDocumentSz(r)
	return
}

func ReadDocumentsSz(r io.Reader) (nBytes int32, ms []bson.M) {
	for {
		nb, m := ReadDocumentSz(r)
		if m == nil {
			break
		}
		ms = append(ms, m)
		nBytes += nb
	}
	return
}

func ReadDocuments(r io.Reader) (ms []bson.M) {
	_, ms = ReadDocumentsSz(r)
	return
}

func ToJson(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("{\"error\":%s}", err.Error())
	}
	return string(b)
}

func isClosedErr(err error) bool {
	if e, ok := err.(*net.OpError); ok {
		if e.Err.Error() == "use of closed network connection" {
			return true
		}
	}
	return false
}

func currentTime() string {
	layout := "2006/01/02-15:04:05.000000"
	return time.Now().Format(layout)
}
