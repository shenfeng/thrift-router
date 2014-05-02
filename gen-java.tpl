package cn.techwolf.data;

import cn.techwolf.data.gen.*;
import org.apache.thrift.TException;

import java.util.List;

/**
 * Created by feng on 3/17/14.
 *
 *
 */

// Do not edit this file, this file is generated in {{time}}

public class DataClients {
    public static final int TRY_COUNT = 2;
    private final ClientsPool pool;

    // TODO ips should be zk's ips
    public DataClients(String ips) {
        String[] parts = ips.split(",");
        if (parts.length % 2 != 0) {
            throw new IllegalArgumentException("两个IP, 第1个为Auto Complete Server IP, 第2个为搜索Server IP, 中间用'，'分割");
        }
        pool = new ClientsPool(ips.split(","));
    }

    public DataClients() {
        // For development only
        this("192.168.1.251:6666,192.168.1.251:6666");
    }


    /**
     * @param source 待提示源
     * @param input  用户输入
     * @param limit  结果个数
     * @return 提示结果
     * @throws ClientException
     */
    public AcResp autoComplete(AcSource source, String input, int limit) throws ClientException {
        AcReq req = new AcReq(source.toString().toLowerCase(), input, limit);
        return autoComplete(req);
    }

    /**
     * @param reqs
     * @return
     * @throws ClientException
     */
    public List<JobCountReq> jobcount(List<JobCountReq> reqs) throws ClientException {

        ThriftClient client = null;
        Exception e = null;

        for (int i = 0; i < TRY_COUNT; i++) {
            try {
                client = pool.getClient(client);
                List<Integer> counts = client.client.jobcount(reqs);

                int ridx = 0;
                for (JobCountReq req : reqs) {
                    req.setCount(counts.get(ridx));
                    ridx++;
                }
                pool.returnClient(client);

                return reqs;
            } catch (TException e1) {
                e = e1;
                if (client != null) {
                    pool.returnBrokenClient(client);
                }
                // retry
            }
        }
        throw new ClientException(e);
    }




{% for s in services %}
   // auto generated code, do not edit
   public {{s.resp}} {{s.fname}}({{s.arg.type}} {{s.arg.name}}) throws ClientException {
        ThriftClient client = null;
        Exception e = null;

        for (int i = 0; i < TRY_COUNT; i++) {
            try {
                client = pool.getClient(client);
                {{s.resp}} resp = client.client.{{s.name}}({{s.arg.name}});
                pool.returnClient(client);
                return resp;
            } catch (TException e1) {
                e = e1;
                if (client != null) {
                    pool.returnBrokenClient(client);
                }
                // retry
            }
        }
        throw new ClientException(e);
    }
{% endfor %}
}
