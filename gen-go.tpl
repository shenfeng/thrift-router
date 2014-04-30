package main

import (
	"fmt"
	"data"
	"errors"
	"thrift"
	"crypto/md5"
)

func readThrift(input []byte, t TType) error {
	prot := thrift.NewTRBinaryProtocol(input[4:])
	if _, mTypeId, _, err := prot.ReadMessageBegin(); err != nil {
		return err
	} else if mTypeId == thrift.EXCEPTION {
		return errors.New("uppper exception")
	}
	return t.Read(prot)
}

func md5Sum(t TType) []byte {
	oprot := thrift.NewTWBinaryProtocol(16 * 1024)
	t.Write(oprot)
	data := oprot.Bytes()

	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

func NewHooker(fname string) IHooker {
	switch fname {
	{% for name, upper in names %}
	case "{{name}}":
		return &{{upper}}Hook{}
	{% endfor %}
	default:
		panic("unknow " + fname)
	}
}

{% for lower, name in names %}

type {{name}}Hook struct{}
func (p *{{name}}Hook) DecodeReq(s *Server, input []byte, seqid int32) (*TRequest, error) {
	r := data.New{{name}}Args()
	if err := readThrift(input, r); err != nil {
		return nil, err
	} else {
		return &TRequest{
			bytes: input,
			obj: r,
			cacheKey: fmt.Sprintf("{{lower}}-%x", md5Sum(r)),
		}, nil
	}
}
func (p *{{name}}Hook) Log(r *Reducer) {}
func (p *{{name}}Hook) Reduce(input []byte, prev TType, r *Reducer) (bool, TType, error) {
	res := data.New{{name}}Result()
	if err := readThrift(input, res); err == nil {
		return true, res, nil
	} else {
		return false, nil, err
	}
}

{% endfor %}
