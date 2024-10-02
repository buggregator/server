<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller;

use App\Application\AppVersion;
use App\Application\Auth\AuthSettings;
use App\Application\HTTP\Response\JsonResource;
use App\Application\HTTP\Response\ResourceInterface;
use App\Application\Ide\UrlTemplate;
use App\Application\Client\Settings;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Router\Annotation\Route;

final class SettingsAction
{
    #[Route(route: 'settings', methods: ['GET'], group: 'api_guest')]
    public function __invoke(
        EnvironmentInterface $env,
        AuthSettings $settings,
        UrlTemplate $ideUrl,
        AppVersion $appVersion,
        Settings $clientSettings,
    ): ResourceInterface {
        $supportedEvents = \array_filter(\explode(',', $clientSettings->supportedEvents));

        return new JsonResource([
            'auth' => [
                'enabled' => $settings->enabled,
                'login_url' => (string) $settings->loginUrl,
            ],
            'ide' => [
                'url_template' => $ideUrl->template,
            ],
            'version' => $appVersion->version,
            'client' => [
                'events' => $supportedEvents,
            ],
        ]);
    }
}
