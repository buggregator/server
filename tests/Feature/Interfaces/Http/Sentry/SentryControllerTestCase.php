<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http\Sentry;

use Sentry\Client;
use Sentry\Options;
use Sentry\Serializer\PayloadSerializer;
use Sentry\Severity;
use Sentry\State\Scope;
use Tests\App\Sentry\FakeTransport;
use Tests\Feature\Interfaces\Http\ControllerTestCase;

abstract class SentryControllerTestCase extends ControllerTestCase
{
    public function getClient(): Client
    {
        $options = new Options();
        return new Client($options, new FakeTransport(new PayloadSerializer($options)));
    }

    public function makeEventPayload(string $message, ?Severity $level = null): string
    {
        $client = $this->getClient();

        $level = $level ?? Severity::info();

        $scope = new Scope();

        $eventId = $client->captureMessage(message: $message, level: $level, scope: $scope);

        return $client->getTransport()->findEvent($eventId);
    }
}
