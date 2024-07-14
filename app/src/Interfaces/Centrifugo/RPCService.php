<?php

declare(strict_types=1);

namespace App\Interfaces\Centrifugo;

use RoadRunner\Centrifugo\Request\RequestInterface;
use RoadRunner\Centrifugo\Request\RPC;
use App\Application\Domain\Assert;
use Psr\Http\Message\ServerRequestFactoryInterface;
use Psr\Http\Message\ServerRequestInterface;
use RoadRunner\Centrifugo\Payload\RPCResponse;
use Spiral\Filters\Exception\ValidationException;
use Spiral\Http\Http;
use Spiral\RoadRunnerBridge\Centrifugo\ServiceInterface;

final readonly class RPCService implements ServiceInterface
{
    public function __construct(
        private Http $http,
        private ServerRequestFactoryInterface $requestFactory,
    ) {}

    public function handle(RequestInterface $request): void
    {
        \assert($request instanceof RPC);

        $result = [];

        try {
            $response = $this->http->handle(
                $this->createHttpRequest($request),
            );

            $result = \json_decode((string) $response->getBody(), true);
            $result['code'] = $response->getStatusCode();
        } catch (ValidationException $e) {
            $result['code'] = $e->getCode();
            $result['errors'] = $e->errors;
            $result['message'] = $e->getMessage();
        } catch (\Throwable $e) {
            $result['code'] = $e->getCode();
            $result['message'] = $e->getMessage();
        }

        try {
            $request->respond(
                new RPCResponse(
                    data: $result,
                ),
            );
        } catch (\Throwable $e) {
            $request->error((int) $e->getCode(), $e->getMessage());
        }
    }

    public function createHttpRequest(RPC $request): ServerRequestInterface
    {
        Assert::string($request->method, 'Invalid method');
        Assert::true(\str_contains($request->method, ':'), 'Invalid method format');

        /** @psalm-suppress PossiblyUndefinedArrayOffset */
        [$method, $uri] = \explode(':', $request->method, 2);
        $method = \strtoupper($method);

        $httpRequest = $this->requestFactory->createServerRequest(\strtoupper($method), \ltrim($uri, '/'))
            ->withHeader('Content-Type', 'application/json');

        $data = $request->getData();

        $token = $data['token'] ?? null;
        unset($data['token']);

        $httpRequest = match ($method) {
            'GET', 'HEAD' => $httpRequest->withQueryParams($data),
            'POST', 'PUT', 'DELETE' => $httpRequest->withParsedBody($data),
            default => throw new \InvalidArgumentException('Unsupported method'),
        };

        if (\is_string($token) && ($token !== '' && $token !== '0')) {
            $httpRequest = $httpRequest->withHeader('X-Auth-Token', $token);
        }

        return $httpRequest;
    }
}
