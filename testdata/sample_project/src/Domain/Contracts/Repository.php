<?php

namespace App\Domain\Contracts;

interface Repository
{
    public function findById(int $id): mixed;
    public function findAll(): array;
    public function save(mixed $entity): void;
    public function delete(int $id): void;
}
