<?php

namespace App\Tests;

use App\Repo\UserRepo;

class UserRepoTest
{
    public function testFind(): void
    {
        UserRepo::find(1);
        UserRepo::find(2);
        UserRepo::find(3);
        helperInTest();
    }
}

function helperInTest(): void {}
