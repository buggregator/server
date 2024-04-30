<?php

declare(strict_types=1);

namespace Interfaces\Console;

use Spiral\Testing\Attribute\Env;
use Tests\TestCase;

final class RegisterModulesTest extends TestCase
{
    #[Env('PERSISTENCE_DRIVER', 'memory')]
    public function testCommand(): void
    {
        $this->spyConsole(function () {
            $this->getConsole()->run('register:modules');
        }, ['register:modules'])
            ->assertCommandNotRun('migrate')
            ->assertCommandRun('webhooks:register')
            ->assertCommandRun('metrics:declare')
            ->assertCommandRun('projects:register');
    }

    #[Env('PERSISTENCE_DRIVER', 'mongodb')]
    public function testCommandWithMongoDriver(): void
    {
        $this->spyConsole(function () {
            $this->getConsole()->run('register:modules');
        }, ['register:modules'])
            ->assertCommandNotRun('migrate')
            ->assertCommandRun('webhooks:register')
            ->assertCommandRun('metrics:declare')
            ->assertCommandRun('projects:register');
    }

    #[Env('PERSISTENCE_DRIVER', 'database')]
    public function testCommandWithDatabaseDriver(): void
    {
        $this->spyConsole(function () {
            $this->getConsole()->run('register:modules');
        }, ['register:modules'])->assertCommandRun('migrate')
            ->assertCommandRun('webhooks:register')
            ->assertCommandRun('metrics:declare')
            ->assertCommandRun('projects:register');
    }
}
