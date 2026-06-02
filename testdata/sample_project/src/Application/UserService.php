<?php

namespace App\Application;

use App\Application\OrderService;
use App\Domain\User;
use App\Domain\UserRepository;
use App\Domain\Enums\UserStatus;
use App\Domain\ValueObjects\Email;

class UserService
{
    private UserRepository $userRepository;
    private OrderService $orderService;

    public function __construct(UserRepository $userRepository, OrderService $orderService)
    {
        $this->userRepository = $userRepository;
        $this->orderService = $orderService;
    }

    public function getUserById(int $id): ?User
    {
        return $this->userRepository->findById($id);
    }

    public function getUserWithOrders(int $id): array
    {
        $user = $this->userRepository->findById($id);
        if ($user === null) {
            return [];
        }
        // Circular: UserService calls OrderService
        $orders = $this->orderService->getOrdersByUser($id);
        $totalSpent = $this->orderService->getTotalSpentByUser($id);
        return ['user' => $user->jsonSerialize(), 'orders' => $orders, 'totalSpent' => $totalSpent];
    }

    public function activateUser(int $id): User
    {
        $user = $this->userRepository->findById($id);
        if ($user === null) {
            throw new \RuntimeException("User {$id} not found");
        }
        $user->activate();
        $this->userRepository->save($user);
        return $user;
    }

    public function suspendUser(int $id, string $reason): void
    {
        $user = $this->userRepository->findById($id);
        if ($user === null) {
            throw new \RuntimeException("User not found");
        }
        $user->suspend();
        $this->userRepository->save($user);
    }

    public function promoteToAdmin(int $id): User
    {
        $user = $this->userRepository->findById($id);
        if ($user === null) {
            throw new \RuntimeException("User {$id} not found");
        }
        $user->promote(User::ADMIN_ROLE);
        $this->userRepository->save($user);
        return $user;
    }

    public function getActiveUsersCount(): int
    {
        return $this->userRepository->countByStatus(UserStatus::Active);
    }

    public function findByEmail(string $email): ?User
    {
        return $this->userRepository->findByEmail($email);
    }
}
