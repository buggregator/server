<?php

declare(strict_types=1);

namespace Tests\Feature\Interfaces\Http;

use Tests\App\Http\HttpFaker;
use Tests\DatabaseTestCase;

abstract class ControllerTestCase extends DatabaseTestCase
{
    protected HttpFaker $http;

    protected function setUp(): void
    {
        parent::setUp();

        $this->http = new HttpFaker($this->fakeHttp());
    }
}
