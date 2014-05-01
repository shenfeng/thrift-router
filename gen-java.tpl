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
