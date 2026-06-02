<?php

namespace App\Domain;

use App\Domain\Contracts\Repository;
use App\Domain\Enums\OrderStatus;

interface OrderRepository extends Repository
{
    public function findByUserId(int $userId): array;
    public function findByStatus(OrderStatus $status): array;
    public function findPendingOlderThan(\DateTime $date): array;
    public function getTotalRevenue(): int;
    public function countByUser(int $userId): int;
}
