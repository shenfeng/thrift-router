#encoding: utf8
import sys

sys.path.append('../gen-py')
from data.DataService import Client
from data.ttypes import JobSearchReq
from thrift.transport import TSocket, TTransport
from thrift.protocol import TBinaryProtocol
import random, multiprocessing, argparse
from time import time

parser = argparse.ArgumentParser(description='Thrift proxy tester')
parser.add_argument('--loop', type=int, default=1000, help='Number of request per process')
parser.add_argument('--process', type=int, default=4, help='Number of process')
parser.add_argument('--port', type=int, default=8401, help='Server port')
parser.add_argument('--server', type=str, default="localhost", help='Server Addr')
args = parser.parse_args()


def getClient():
    transport = TSocket.TSocket(args.server, args.port)
    transport.open()
    trans = TTransport.TFramedTransport(transport)
    protocol = TBinaryProtocol.TBinaryProtocolAccelerated(trans)
    client = Client(protocol)
    return client


ids = range(6000, 6000 + 6000)
random.shuffle(ids)
up = JobSearchReq(query='研发中心助理', city='', limit=10, offset=0)
# print "ids", ids[:10], "sorted", getClient().jobSearch(up)


def get_stats(times):
    times = sorted(times)
    pos_50 = times[int(len(times) * 0.50)]
    pos_80 = times[int(len(times) * 0.80)]
    pos_95 = times[int(len(times) * 0.95)]
    pos_99 = times[int(len(times) * 0.99)]
    return "50%%=%.4fms, avg=%.4fms 80%%=%.4fms, 95%%=%.4fms, 99%%=%.4fms, max=%.4fms" % (
        pos_50, sum(times) / len(times), pos_80, pos_95, pos_99, max(times))


def target(id):
    times = []
    failed = 0

    service = getClient()
    for i in range(1, args.loop):
        sort_ids = ids[:random.randint(500, len(ids))]
        if len(sort_ids) > 3000:
            sort_ids = ids[:random.randint(500, len(ids))]
        expected = sorted(sort_ids)[:64]
        random.shuffle(sort_ids)
        up.ids = sort_ids
        start = time()
        try:
            r = service.jobSearch(up)
        except Exception as e:
            failed += 1
        times.append((time() - start) * 1000)
    print id, "failed", failed, get_stats(sorted(times))


ps = []
for id in range(args.process):
    p = multiprocessing.Process(target=target, args=(id,))
    p.start()
    ps.append(p)

for p in ps:
    p.join()
