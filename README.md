# mgosniff - MongoDB Wire Protocol Analysis Tools

Reference: [MongoDB Wire Protocol](https://docs.mongodb.org/manual/reference/mongodb-wire-protocol/)

### Introduction

Different from [mongosniff](https://docs.mongodb.org/manual/reference/program/mongosniff/), `mgosniff` acted as a tcp proxy between client and mongodb server.  all client request treffic and mongodb server reply traffic will go through `mgosniff`, `mgosniff` understand the protocal, so it will do analyse and show what's is going on between client and server in human-readable way.


