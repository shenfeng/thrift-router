__author__ = 'feng'

import re
from jinja2 import Template

# thrift -gen go data.thrift thrift_import=thrift

template = Template('''
package main

func readThrift(input []byte, t TType) error {
	prot := thrift.NewTRBinaryProtocol(input[4:])
	if _, mTypeId, _, err := prot.ReadMessageBegin(); err != nil {
		return err
	} else if mTypeId == thrift.EXCEPTION {
		return errors.New("uppper exception")
	}
	return t.Read(prot)
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

{% for _, name in names %}

type {{name}}sHook struct{}
func (p *{{name}}sHook) DecodeReq(s *Server, input []byte, seqid int32) ([]byte, TType, error) {
	r := New{{name}}sArgs()
	if err := readThrift(input, r); err != nil {
		return nil, nil, err
	} else {
		return input, r, nil
	}
}
func (p *{{name}}sHook) Log(r *Reducer) {}
func (p *{{name}}sHook) Reduce(input []byte, prev TType, r *Reducer, stra *TStrategy) (bool, TType, error) {
	res := New{{name}}sResult()
	if err := readThrift(input, res); err == nil {
		return true, res, nil
	} else {
		return false, nil, err
	}
}

{% endfor %}

''')


def first_upper(name):
    return name[0].upper() + name[1:]


def gen_hooks(file, out):
    FIND_SERVICE, FIND_START, FIND_NAME, FIND_END = range(4)
    state = FIND_SERVICE

    names = []

    for line in open(file):
        if state == FIND_SERVICE:
            if 'service ' in line:
                state = FIND_START
                if '{' in line:
                    state = FIND_NAME
        elif state == FIND_START:
            if '{' in line:
                state = FIND_NAME
        elif state == FIND_NAME:
            m = re.search("\s+(\w+)\s?\(", line)
            if m:
                # names.append(m.group(1))

                names.append((m.group(1), first_upper(m.group(1))))

                # gen_hook(m.group(1), out)

            if '}' in line:
                state = FIND_SERVICE

    # out = open(out, 'w')
    print template.render(names=names)


if __name__ == "__main__":
    gen_hooks('/Users/feng/gocode/src/github.com/kylentechwolf/engine/data.thrift', '/tmp/what.go')


