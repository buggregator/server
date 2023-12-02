<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http;

use Tests\App\Http\HttpFaker;
use Tests\TestCase;

abstract class ControllerTestCase extends TestCase
{
    protected HttpFaker $http;

    protected function setUp(): void
    {
        parent::setUp();

        $this->http = new HttpFaker($this->fakeHttp(), $this);
    }
}
