<?php

declare(strict_types=1);
require __DIR__ . '/vendor/autoload.php';

use OpenTelemetry\Contrib\Otlp\OtlpHttpTransportFactory;
use OpenTelemetry\Contrib\Otlp\SpanExporter;
use OpenTelemetry\SDK\Common\Attribute\Attributes;
use OpenTelemetry\SDK\Common\Time\ClockFactory;
use OpenTelemetry\SDK\Trace\SpanProcessor\BatchSpanProcessor;
use OpenTelemetry\SDK\Trace\TracerProvider;
use OpenTelemetry\SDK\Resource\ResourceInfo;
use OpenTelemetry\SDK\Resource\ResourceInfoFactory;
use OpenTelemetry\SDK\Trace\Span;
use OpenTelemetry\API\Trace\StatusCode;
use OpenTelemetry\Context\Context;


$transport = (new OtlpHttpTransportFactory())->create('http://tracing-analysis-dc-hz.aliyuncs.com/adapt_eg0azz190g@cf15eacf3a67cc0_eg0azz190g@53df7ad2afe8301/api/otlp/traces', 'application/json');
$exporter = new SpanExporter($transport);
$resource = ResourceInfoFactory::merge(ResourceInfo::create(Attributes::create(['service.name' => 'otlp-php', 'host.name' => gethostname()])), ResourceInfoFactory::defaultResource());


echo 'Starting OTLP+json example';

$tracerProvider = new TracerProvider(
    new BatchSpanProcessor(
        $exporter,
        ClockFactory::getDefault()
    ),
    null,
    $resource,
);
$tracer = $tracerProvider->getTracer('otlp-demo-tracer');

OpenTelemetry\Instrumentation\hook(
    DemoClass::class,
    'run',
    static function (DemoClass $demo, array $params, string $class, string $function, ?string $filename, ?int $lineno) use ($tracer) {
        $tracer->spanBuilder($class)
            ->startSpan()
            ->activate();
    },
    static function (DemoClass $demo, array $params, $returnValue, ?Throwable $exception) use ($tracer) {
        $scope = Context::storage()->scope();
        $scope?->detach();
        $span = Span::fromContext($scope->context());
        $exception && $span->recordException($exception);
        $span->setStatus($exception ? StatusCode::STATUS_ERROR : StatusCode::STATUS_OK);
        $span->end();
    }
);

class DemoClass
{
    public function run(): void
    {
        echo "running";
    }
}


// $root = $span = $tracer->spanBuilder('root')->startSpan();
// // do some work here
// $root->end();

$demo_fun = new DemoClass();
$demo_fun->run();

echo PHP_EOL . 'OTLP+json example complete!  ';
echo PHP_EOL;
echo date('Y-m-d H:i:s');
$tracerProvider->shutdown();
