java -javaagent:opentelemetry-javaagent.jar \
     -Dotel.service.name=otlp-spring \
     -Dotel.traces.exporter=otlp \
     -Dotel.exporter.otlp.headers=Authentication=<你的阿里云 grpc token> \
     -Dotel.exporter.otlp.endpoint=http://tracing-analysis-dc-hz.aliyuncs.com:8090 \
     -jar otlp-0.0.1-SNAPSHOT.jar