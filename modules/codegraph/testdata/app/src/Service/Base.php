<?php

namespace App\Service;

abstract class Base
{
    protected function boot(): void
    {
        helper2();
    }
}

function helper2(): void {}
