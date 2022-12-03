<?php

declare(strict_types=1);

namespace Tests\Feature\Controller;

use Tests\TestCase;

class HomeControllerTest extends TestCase
{
    public function testDefaultActionWorks(): void
    {
        $this
            ->fakeHttp()
            ->get('/')
            ->assertOk()
            ->assertBodyContains('Welcome to Spiral Framework');
    }

    public function testDefaultActionWithRuLocale(): void
    {
        $this
            ->fakeHttp()
            ->withHeader('accept-language', 'ru')
            ->get('/')
            ->assertOk()
            ->assertBodyContains('Вас приветствует Spiral Framework');
    }

    public function testInteractWithConsole(): void
    {
        $output = $this->runCommand('views:reset');

        $this->assertStringContainsString('cache', $output);
    }
}
