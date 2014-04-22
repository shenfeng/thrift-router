<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1">
    <title>Thrift Router</title>
  </head>
  <style>
    html, body { margin:0; padding:0; }
    table { width: 800px; }
    #page-wrap { width: 800px; margin: 0 auto; }
    body { font-family: monospace; }
    .server { border: 1px; }
    ul { list-style: none; padding: 0;}
    tr:nth-child(2n) { background: #eee;}
    td { padding: 3px; }
    .servers > li { padding-bottom: 20px; border-bottom: 1px solid #fac;}
    .server { padding-bottom: 10px; }
    h2, h3 { text-align: center; }
    tr td:first-child { width: 190px; }
    caption { font-weight: bold; }
    #debug-link { float: right; margin-right: 2em;}
    .strategy { padding-bottom: 10px; }
  </style>
  <body>
    <div id="page-wrap">
      <p id="debug-link"><a href="/debug/pprof">debug</a></p>
      <h2>Thrift router & cacher</h2>
      <ul class="servers">
        <li>
          <table class="root">
            <tr><td>Startup time</td><td>{{.start}}</td></tr>
            <tr><td>Running</td><td>{{.time}}</td></tr>
            <tr><td>Total</td><td>{{.hits}}</td></tr>
            <tr><td>Overall QPS</td><td>{{.qps}}</td></tr>
            <tr><td>Fails</td><td>{{.fails}}</td></tr>
            <tr><td>Latencies</td><td>{{.latencies}}</td></tr>
            <tr><td>Avg Latency</td><td>{{.avgLatency}}</td></tr>
          </table>
        </li>
        <li>
          <h3>Traffic Routing Strategy</h3>
          {{range .services}}
          <ul>
            <li>
              <h4>{{ .name }}</h4>
              {{range .strategies}}
              <table class="strategy">
                <caption>Strategy: {{ .name }}, Policy: {{.policy}}, Copy traffic: {{.copy}}</caption>
                <tr><td>Traffic</td><td>{{ .traffic }}; hits: {{ .hits }}; fails: {{ .fails }}; qps: {{ .qps}}; avg latency: {{ .avgLatency }}</td></tr>
                <tr><td>Cache</td><td>cache {{.cachetime}}s. {{ .cache }}</td></tr>
                <tr><td>Models</td><td>{{ .models }}</td></tr>
                <tr><td>Partials</td><td>partials: {{ .partials }}, Timeouts ({{ .timeout }}ms): {{ .timeouts }}</td></tr>
                <tr><td>Latencies</td><td>{{ .latencies }}</td></tr>
              </table>
              {{end}} <!-- strategies -->
            </li>
          </ul>
          {{end}} <!-- services -->

        <li>
          <h3>Backend Servers</h3>
          {{range .backends}}
          <table class="backend">
            <caption>{{ .name }}</caption>
            <tr><td>Hits</td><td>hits: {{ .hits }}; fails: {{ .fails }}; timeout: {{ .timeouts }}</td></tr>
            <tr><td>Qps</td><td>{{ .qps }}</td></tr>
            <tr><td>Ongoing</td><td>{{ .ongoing }}</td></tr>
            <tr><td>Dead</td> <td>{{ .dead }}</td></tr>
            <tr><td>Avg Latency</td> <td>{{ .avgLatency }}</td></tr>
            <tr><td>Latencies</td> <td>{{ .latencies }}</td></tr>
          </table>
          {{end}}
        </li>
      </ul>
    </div>
  </body>
</html>
