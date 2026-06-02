<?php

namespace App\Domain;

use App\Domain\Contracts\Repository;
use App\Domain\Enums\UserStatus;

interface UserRepository extends Repository
{
    public function findByEmail(string $email): ?User;
    public function findByStatus(UserStatus $status): array;
    public function findActiveUsers(int $limit): array;
    public function countByStatus(UserStatus $status): int;
    public function existsByEmail(string $email): bool;
}
