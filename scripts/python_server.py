import sys

sys.path.append('../gen-py')

import multiprocessing
from data.DataService import Processor
from data.ttypes import JobSearchResp, TinyJob

from thrift.transport import TSocket
from thrift.transport import TTransport
from thrift.protocol import TBinaryProtocol
from thrift.server import TServer
from thrift.server import TNonblockingServer

import time, random, argparse

parser = argparse.ArgumentParser(description='Thrift proxy tester')
parser.add_argument('--ports', type=str, default="7070,7071,7072", help='Ports to listen to')
parser.add_argument('--degradation', type=str, default="true", help='Is Degradation server')
args = parser.parse_args()

ports = map(int, args.ports.split(','))


class UserStorageHandler:
    def __init__(self):
        self.users = {}


    def jobSearch(self, req):
        jobs = []
        for i in range(random.randint(4)):
            jobs.append(TinyJob(id=i + 1000, title="this is a test",
                                company="this is a company", salary='1212'))

        return JobSearchResp(count=100, jobs=jobs, counters={}, milliseconds=100)


def target(port):
    print "Listen on port:", port, ", Degradation:", args.degradation == 'true'
    processor = Processor(UserStorageHandler())
    transport = TSocket.TServerSocket(port=port)

    # tfactory = TTransport.TBufferedTransportFactory()
    tfactory = TTransport.TFramedTransportFactory()
    pfactory = TBinaryProtocol.TBinaryProtocolFactory()

    # server = TServer.TSimpleServer(processor, transport, tfactory, pfactory)
    # server = TServer.TThreadPoolServer(processor, transport, tfactory, pfactory, daemon=True)
    prof = TBinaryProtocol.TBinaryProtocolAcceleratedFactory()
    server = TNonblockingServer.TNonblockingServer(processor, transport, prof)
    server.setNumThreads(6)
    server.serve()
    print 'done.'


ps = []
for port in ports:
    p = multiprocessing.Process(target=target, args=(port,))
    p.start()
    ps.append(p)

for p in ps:
    p.join()
