<?php

declare(strict_types=1);

namespace App\Interfaces\Http\Controller;

use App\Application\Auth\AuthSettings;
use App\Application\HTTP\Response\JsonResource;
use App\Application\HTTP\Response\ResourceInterface;
use Spiral\Boot\EnvironmentInterface;
use Spiral\Router\Annotation\Route;

final class SettingsAction
{
    #[Route(route: 'settings', methods: ['GET'], group: 'api_guest')]
    public function __invoke(EnvironmentInterface $env, AuthSettings $settings): ResourceInterface
    {
        return new JsonResource([
            'auth' => [
                'enabled' => $settings->enabled,
                'login_url' => (string) $settings->loginUrl,
            ],
            'version' => $env->get('APP_VERSION', 'dev'),
        ]);
    }
}
