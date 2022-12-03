<?php

declare(strict_types=1);

namespace App\Application\Service\ErrorHandler;

use Spiral\Exceptions\ExceptionHandler;
use Spiral\Exceptions\Renderer\ConsoleRenderer;

/**
 * -----------------------------------------------------
 * In this file, you can modify the exception handling
 * -----------------------------------------------------
 */
final class Handler extends ExceptionHandler
{
    protected function bootBasicHandlers(): void
    {
        parent::bootBasicHandlers();
        $this->addRenderer(new ConsoleRenderer());
    }
}
