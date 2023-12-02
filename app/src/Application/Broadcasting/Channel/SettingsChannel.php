<?php

declare(strict_types=1);

namespace App\Application\Broadcasting\Channel;

final class SettingsChannel extends Channel
{
    public function __construct()
    {
        parent::__construct('settings');
    }
}
