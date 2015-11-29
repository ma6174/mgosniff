package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sbunce/bson"
)

func ReadInt32(r io.Reader) (n int32) {
	err := binary.Read(r, binary.LittleEndian, &n)
	if err != nil {
		panic(err)
	}
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

func ReadDocument(r io.Reader) *bson.Map {
	m, err := bson.ReadMap(r)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		panic(err)
	}
	return &m
}

func ReadDocuments(r io.Reader) (ms []*bson.Map) {
	for {
		m := ReadDocument(r)
		if m == nil {
			break
		}
		ms = append(ms, m)
	}
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
