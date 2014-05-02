# encoding: utf8
__author__ = 'feng'

import re
from jinja2 import Template

import argparse
import datetime
parser = argparse.ArgumentParser(description="Generate go src file")
parser.add_argument('--file', type=str, default="data.thrift", help='Thrift definition file')
parser.add_argument('--mode', type=str, default="gen-java", help='gen-go | gen-java | check')
args = parser.parse_args()


# thrift -gen go:thrift_import=thrift data.thrift


FIND_SERVICE, FIND_START, FIND_NAME, FIND_END = range(4)


def first_upper(name):
    return name[0].upper() + name[1:]


class ThriftService(object):
    def to_java_type(self, t):
        m = {
            'list<': 'List<',
            'i32': 'Integer',
            'i64': 'Long',
            'string': 'String',
        }

        for k, v in m.items():
            t = t.replace(k, v)

        return t

    def parse_args(self, args):
        results = []

        for arg in args.split(','):
            if not arg:
                continue

            parts = arg.split(':', 1)

            if len(parts) == 2:
                arg = parts[1].strip()
            else:
                arg = arg.strip()

            argtype, name = re.split('\s+', arg)
            results.append({
                'type': self.to_java_type(argtype),
                'name': name
            })

        return results


    def __init__(self, line):
        line = line.strip()
        self.resp, self.name, _ = re.split('\s+|\(', line, 2)

        self.Name = first_upper(self.name)
        args = re.search('\((.*)\)', line).group(1)
        self.args = self.parse_args(args)

        if len(self.args) == 1:
            self.arg = self.args[0]
        elif len(self.args) == 0:
            self.arg = {'type': '', 'name': ''}

        self.resp = self.to_java_type(self.resp)


    def __str__(self):
        return str(self.__dict__)

    def __repr__(self):
        return str(self.__dict__)


def get_service_lines(thrift):
    state = FIND_SERVICE

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
                # yield line.strip()

                yield ThriftService(line.strip())

                # names.append(m.group(1))

                # names.append((m.group(1), first_upper(m.group(1))))

                # gen_hook(m.group(1), out)

            if '}' in line:
                state = FIND_SERVICE


def check_config(f):
    import json

    # json.

    print json.dumps(json.loads(open(f).read()), indent=2)


def gen_go_hooks():
    names = []

    for ts in get_service_lines(args.file):
        names.append((ts.name, ts.Name))

    py_template = Template(open('gen-go.tpl').read())
    print py_template.render(names=names)


def gen_java_hooks():
    services = []

    rename = {
        'mobilePopular': 'mobilePopularJobs',
        'autocomplete': 'autoComplete'
    }
    for ts in get_service_lines(args.file):
        ts.fname = rename.get(ts.name, ts.name)

        if len(ts.args) <= 1 and ts.name not in ('jobcount', 'findSimilaryJobs'):
            services.append(ts)

            # print ts
    java_template = Template(open('gen-java.tpl').read().decode('utf8'))
    print java_template.render(services=services,
                               time=datetime.datetime.now()).encode('utf8')


if __name__ == "__main__":
    if args.mode == 'gen-go':
        gen_go_hooks()
    elif args.mode == 'gen-java':
        gen_java_hooks()
    elif args.mode == 'check':
        check_config(args.file)


