__author__ = 'feng'

import re
from jinja2 import Template

import argparse

parser = argparse.ArgumentParser(description="Generate go src file")
parser.add_argument('--file', type=str, default="data.thrift", help='Thrift definition file')
parser.add_argument('--mode', type=str, default="gen", help='Generate go source file')
args = parser.parse_args()


# thrift -gen go:thrift_import=thrift data.thrift

template = Template(open('gen-go.tpl').read())
FIND_SERVICE, FIND_START, FIND_NAME, FIND_END = range(4)


def first_upper(name):
    return name[0].upper() + name[1:]


def gen_hooks(thrift):
    state = FIND_SERVICE
    names = []

    for line in open(thrift):
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


def check_config(f):
    import json

    # json.

    print json.dumps(json.loads(open(f).read()), indent=2)


if __name__ == "__main__":
    if args.mode == 'gen':
        gen_hooks(args.file)
    elif args.mode == 'check':
        check_config(args.file)


