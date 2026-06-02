<?php

namespace App\Domain\Contracts;

interface Cacheable
{
    public function getCacheKey(): string;
    public function getCacheTtl(): int;
    public function clearCache(string $key): void;
}
