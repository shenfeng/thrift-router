package main

import (
	"encoding/binary"
	// "log"
	"thrift"
)

func binaryProtocolEncode(r *Reducer, t TType) []byte {
	// oprot := &thrift.TRBinaryProtocol{buffer: make([]byte, bufferSize), strictWrite: true}
	oprot := thrift.NewTWBinaryProtocol(16 * 1024)
	oprot.WriteMessageBegin(r.fname, thrift.REPLY, r.seqId)
	t.Write(oprot)
	oprot.WriteMessageEnd()
	data := oprot.Bytes()

	buffer := make([]byte, len(data)+4)
	binary.BigEndian.PutUint32(buffer, uint32(len(data)))
	copy(buffer[4:], data)
	// log.Println("encode", len(buffer))
	return buffer
}

func formatError(fname string, seqId, exceptionId int32, err error) []byte {
	buffer := thrift.NewTMemoryBuffer()
	trans := thrift.NewTFramedTransport(buffer)
	oprot := thrift.NewTBinaryProtocolTransport(trans)
	a := thrift.NewTApplicationException(exceptionId, err.Error())
	oprot.WriteMessageBegin(fname, thrift.EXCEPTION, seqId)
	a.Write(oprot)
	oprot.WriteMessageEnd()
	oprot.Flush()
	return buffer.Bytes()
}

func copyAndfixSeqId(buffer []byte, seqid int32) []byte {
	// thrift protol use seqid. Python client does not increament or check it,
	// Go client check it

	// 4 bytes, total length
	// 4 bytes, VERSION_1 | type
	// 4 bytes, name length, name
	// 4 bytes, seq_id
	t := make([]byte, len(buffer))
	copy(t, buffer)
	buffer = t

	nameLen := int32(binary.BigEndian.Uint32(buffer[8:]))

	// log.Println(nameLen, "---------------", len(buffer))
	// log.Println(string(buffer[09:12+nameLen]))
	binary.BigEndian.PutUint32(buffer[12+nameLen:], uint32(seqid))
	return buffer
}
