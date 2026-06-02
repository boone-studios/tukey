<?php

namespace App\Domain\Contracts;

interface Eventable
{
    public function getEventName(): string;
    public function getPayload(): array;
}
