# mgosniff - MongoDB Wire Protocol Analysis Tools

Reference: [MongoDB Wire Protocol](https://docs.mongodb.org/manual/reference/mongodb-wire-protocol/)

### Introduction

Different from [mongosniff](https://docs.mongodb.org/manual/reference/program/mongosniff/), `mgosniff` acted as a tcp proxy between client and mongodb server.  all client request treffic and mongodb server reply traffic will go through `mgosniff`, `mgosniff` understand the protocal, so it will do analyse and show what is going on between client and server in human-readable way.

### Install

```
go get github.com/ma6174/mgosniff
```

## Example

##### 1. start mgosniff

```shell
$ mgosniff -h
Usage of mgosniff:
  -d string
    	proxy to dest addr (default "127.0.0.1:27017")
  -l string
    	listen port (default ":7017")
  -v	show version
$ mgosniff
2015/11/29 17:01:45 parser.go:278: mgosniff listen at :7017, proxy to mongodb server 127.0.0.1:27017
```

now mgosniff running at 0.0.0.0:7017

##### 2. connect to mgosniff and do some operation

```shell
$ cat test.js
printjson(db.version())
printjson(db.test.insert([{test1:1},{test2:2}]))
printjson(db.test.find().toArray())
printjson(db.test.remove({test1:1}))
printjson(db.test.update({test2:2},{"$inc":{"test2":1}}))
printjson(db.test.find().toArray())
printjson(db.test.drop())
$ mongo localhost:7017/testdb test.js
MongoDB shell version: 3.0.7
connecting to: localhost:7017/testdb
"3.0.7"
{
	"writeErrors" : [ ],
	"writeConcernErrors" : [ ],
	"nInserted" : 2,
	"nUpserted" : 0,
	"nMatched" : 0,
	"nModified" : 0,
	"nRemoved" : 0,
	"upserted" : [ ]
}
[
	{
		"_id" : ObjectId("565abe9375a6567f7febb464"),
		"test1" : 1
	},
	{
		"_id" : ObjectId("565abe9375a6567f7febb465"),
		"test2" : 2
	}
]
{ "nRemoved" : 1 }
{ "nMatched" : 1, "nUpserted" : 0, "nModified" : 1 }
[ { "_id" : ObjectId("565abe9375a6567f7febb465"), "test2" : 3 } ]
true
```

##### 3. all request and reply showed in mgosniff log

```shell
$ mgosniff
2015/11/29 17:01:45 parser.go:278: mgosniff listen at :7017, proxy to mongodb server 127.0.0.1:27017
2015/11/29 17:05:48 parser.go:226: [127.0.0.1:52117] new client connected
2015/11/29-17:05:48.941042 [127.0.0.1:52117] QUERY id:0 coll:admin.$cmd toskip:0 toret:1 flag:0 query:{"whatsmyuri":1} sel:null
2015/11/29-17:05:48.942173 [127.0.0.1:52117] REPLY to:0 flag:1000 curID:859530373936 from:0 reted:1 docs:{"ok":1,"you":"127.0.0.1:52119"}
2015/11/29-17:05:48.943920 [127.0.0.1:52117] QUERY id:1 coll:admin.$cmd toskip:0 toret:-1 flag:0 query:{"buildinfo":1} sel:null
2015/11/29-17:05:48.944697 [127.0.0.1:52117] REPLY to:1 flag:1000 curID:859530374608 from:0 reted:1 docs:{"OpenSSLVersion":"","allocator":"system","bits":64,"compilerFlags":"-Wnon-virtual-dtor -Woverloaded-virtual -std=c++11 -fPIC -fno-strict-aliasing -ggdb -pthread -Wall -Wsign-compare -Wno-unknown-pragmas -Winvalid-pch -pipe -O3 -Wno-unused-local-typedefs -Wno-unused-function -Wno-unused-private-field -Wno-deprecated-declarations -Wno-tautological-constant-out-of-range-compare -Wno-unused-const-variable -Wno-missing-braces -Wno-inconsistent-missing-override -Wno-potentially-evaluated-expression -Wno-null-conversion -mmacosx-version-min=10.11 -std=c99","debug":false,"gitVersion":"nogitversion","javascriptEngine":"V8","loaderFlags":"","maxBsonObjectSize":16777216,"ok":1,"sysInfo":"Darwin elcapitanvm.local 15.0.0 Darwin Kernel Version 15.0.0: Wed Aug 26 16:57:32 PDT 2015; root:xnu-3247.1.106~1/RELEASE_X86_64 x86_64 BOOST_LIB_VERSION=1_49","version":"3.0.7","versionArray":[3,0,7,0]}
2015/11/29-17:05:48.947665 [127.0.0.1:52117] QUERY id:2 coll:admin.$cmd toskip:0 toret:-1 flag:0 query:{"isMaster":1} sel:null
2015/11/29-17:05:48.951487 [127.0.0.1:52117] REPLY to:2 flag:1000 curID:859531149344 from:0 reted:1 docs:{"ismaster":true,"localTime":1448787948948,"maxBsonObjectSize":16777216,"maxMessageSizeBytes":48000000,"maxWireVersion":3,"maxWriteBatchSize":1000,"minWireVersion":0,"ok":1}
2015/11/29-17:05:48.955595 [127.0.0.1:52117] QUERY id:3 coll:testdb.$cmd toskip:0 toret:-1 flag:0 query:{"documents":[{"_id":"Vlq/7INZSsRmFzB4","test1":1},{"_id":"Vlq/7INZSsRmFzB5","test2":2}],"insert":"test","ordered":true} sel:null
2015/11/29-17:05:48.956259 [127.0.0.1:52117] REPLY to:3 flag:1000 curID:859530377472 from:0 reted:1 docs:{"n":2,"ok":1}
2015/11/29-17:05:48.958240 [127.0.0.1:52117] QUERY id:4 coll:testdb.test toskip:0 toret:0 flag:0 query:{} sel:null
2015/11/29-17:05:48.958764 [127.0.0.1:52117] REPLY to:4 flag:1000 curID:859530378112 from:0 reted:2 docs:[{"_id":"Vlq/7INZSsRmFzB4","test1":1},{"_id":"Vlq/7INZSsRmFzB5","test2":2}]
2015/11/29-17:05:48.960042 [127.0.0.1:52117] QUERY id:5 coll:testdb.$cmd toskip:0 toret:-1 flag:0 query:{"delete":"test","deletes":[{"limit":0,"q":{"test1":1}}],"ordered":true} sel:null
2015/11/29-17:05:48.960553 [127.0.0.1:52117] REPLY to:5 flag:1000 curID:859530379024 from:0 reted:1 docs:{"n":1,"ok":1}
2015/11/29-17:05:48.962203 [127.0.0.1:52117] QUERY id:6 coll:testdb.$cmd toskip:0 toret:-1 flag:0 query:{"ordered":true,"update":"test","updates":[{"multi":false,"q":{"test2":2},"u":{"$inc":{"test2":1}},"upsert":false}]} sel:null
2015/11/29-17:05:48.962794 [127.0.0.1:52117] REPLY to:6 flag:1000 curID:859531346320 from:0 reted:1 docs:{"n":1,"nModified":1,"ok":1}
2015/11/29-17:05:48.963472 [127.0.0.1:52117] QUERY id:7 coll:testdb.test toskip:0 toret:0 flag:0 query:{} sel:null
2015/11/29-17:05:48.963868 [127.0.0.1:52117] REPLY to:7 flag:1000 curID:859531347056 from:0 reted:1 docs:{"_id":"Vlq/7INZSsRmFzB5","test2":3}
2015/11/29-17:05:48.964370 [127.0.0.1:52117] QUERY id:8 coll:testdb.$cmd toskip:0 toret:-1 flag:0 query:{"drop":"test"} sel:null
2015/11/29-17:05:48.964970 [127.0.0.1:52117] REPLY to:8 flag:1000 curID:859531347648 from:0 reted:1 docs:{"nIndexesWas":1,"ns":"testdb.test","ok":1}
2015/11/29 17:05:48 parser.go:252: [127.0.0.1:52117] close connection:127.0.0.1:52117
2015/11/29 17:05:48 parser.go:252: [127.0.0.1:52117] close connection:127.0.0.1:27017
```

